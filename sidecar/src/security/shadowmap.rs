use once_cell::sync::Lazy;
use regex::Regex;
use std::collections::HashMap;
use zeroize::Zeroizing;

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum PiiCategory {
    Email,
    Ssn,
    CreditCard,
    Phone,
    IpAddress,
    ApiKey,
    Password,
}

impl std::fmt::Display for PiiCategory {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            PiiCategory::Email => write!(f, "EMAIL"),
            PiiCategory::Ssn => write!(f, "SSN"),
            PiiCategory::CreditCard => write!(f, "CREDIT_CARD"),
            PiiCategory::Phone => write!(f, "PHONE"),
            PiiCategory::IpAddress => write!(f, "IP_ADDRESS"),
            PiiCategory::ApiKey => write!(f, "API_KEY"),
            PiiCategory::Password => write!(f, "PASSWORD"),
        }
    }
}

struct PiiPattern {
    regex: Regex,
    category: PiiCategory,
}

fn is_valid_ipv4(s: &str) -> bool {
    let parts: Vec<&str> = s.split('.').collect();
    parts.len() == 4 && parts.iter().all(|octet| octet.parse::<u8>().is_ok())
}

static PII_PATTERNS: Lazy<Vec<PiiPattern>> = Lazy::new(|| {
    vec![
        // Tier 1: Security Critical
        PiiPattern {
            regex: Regex::new(
                r"\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b|\b\d{4}[\s-]?\d{6}[\s-]?\d{5}\b",
            )
            .unwrap(),
            category: PiiCategory::CreditCard,
        },
        PiiPattern {
            regex: Regex::new(r"\b\d{3}-\d{2}-\d{4}\b").unwrap(),
            category: PiiCategory::Ssn,
        },
        PiiPattern {
            regex: Regex::new(r"(?i)\b(?:sk|pk)[_-][a-zA-Z0-9]{8,}\b").unwrap(),
            category: PiiCategory::ApiKey,
        },
        PiiPattern {
            regex: Regex::new(r"(?i)(?:api[_-]?key|secret[_-]?key)\s*[=:]\s*\S+").unwrap(),
            category: PiiCategory::ApiKey,
        },
        PiiPattern {
            regex: Regex::new(r"(?i)Bearer\s+\S+").unwrap(),
            category: PiiCategory::ApiKey,
        },
        PiiPattern {
            regex: Regex::new(r"(?i)password\s*[=:]\s*\S+").unwrap(),
            category: PiiCategory::Password,
        },
        // Tier 2: Redact by default (run after Tier 1 to avoid CC/SSN overlap)
        PiiPattern {
            regex: Regex::new(r"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}").unwrap(),
            category: PiiCategory::Email,
        },
        PiiPattern {
            regex: Regex::new(r"\b\(?\d{3}\)?[-.\s]\d{3}[-.\s]\d{4}\b").unwrap(),
            category: PiiCategory::Phone,
        },
        // IP: validated at match time
        PiiPattern {
            regex: Regex::new(r"\b(?:\d{1,3}\.){3}\d{1,3}\b").unwrap(),
            category: PiiCategory::IpAddress,
        },
    ]
});

#[derive(Debug, Clone)]
pub struct ShadowMap {
    map: HashMap<String, (Zeroizing<String>, PiiCategory)>,
    counter: u32,
}

impl ShadowMap {
    pub fn new() -> Self {
        Self {
            map: HashMap::new(),
            counter: 0,
        }
    }

    pub fn redact(&mut self, text: &str) -> String {
        let mut result = text.to_string();

        for pattern in PII_PATTERNS.iter() {
            result = pattern
                .regex
                .replace_all(&result, |caps: &regex::Captures| {
                    let matched = caps.get(0).unwrap();
                    let original = matched.as_str();

                    if pattern.category == PiiCategory::IpAddress && !is_valid_ipv4(original) {
                        return original.to_string();
                    }

                    let token = self.next_token();
                    self.map.insert(
                        token.clone(),
                        (
                            Zeroizing::new(original.to_string()),
                            pattern.category.clone(),
                        ),
                    );
                    token
                })
                .into_owned();
        }

        result
    }

    pub fn unredact(&self, text: &str) -> String {
        let mut result = text.to_string();
        for (token, (original, _)) in &self.map {
            result = result.replace(token.as_str(), original.as_str());
        }
        result
    }

    pub fn unredact_token(&self, token: &str) -> Option<&str> {
        self.map.get(token).map(|(original, _)| original.as_str())
    }

    pub fn category(&self, token: &str) -> Option<PiiCategory> {
        self.map.get(token).map(|(_, cat)| cat.clone())
    }

    pub fn map(&self) -> &HashMap<String, (Zeroizing<String>, PiiCategory)> {
        &self.map
    }

    pub fn clear(&mut self) {
        self.map.clear();
        self.counter = 0;
    }

    pub fn contains_pii(text: &str) -> bool {
        for pattern in PII_PATTERNS.iter() {
            if pattern.regex.is_match(text) {
                if pattern.category == PiiCategory::IpAddress {
                    for cap in pattern.regex.captures_iter(text) {
                        if let Some(m) = cap.name("pii") {
                            if is_valid_ipv4(m.as_str()) {
                                return true;
                            }
                        }
                    }
                } else {
                    return true;
                }
            }
        }
        false
    }

    pub fn token_count(&self) -> usize {
        self.map.len()
    }

    fn next_token(&mut self) -> String {
        let token = format!("[PII_{}]", self.counter);
        self.counter += 1;
        token
    }
}

impl Default for ShadowMap {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_is_valid_ipv4() {
        assert!(is_valid_ipv4("192.168.1.100"));
        assert!(is_valid_ipv4("0.0.0.0"));
        assert!(is_valid_ipv4("255.255.255.255"));
        assert!(!is_valid_ipv4("256.1.1.1"));
        assert!(!is_valid_ipv4("1.2.3"));
        assert!(!is_valid_ipv4("0.0.0"));
        assert!(is_valid_ipv4("10.0.0.1"));
    }
}

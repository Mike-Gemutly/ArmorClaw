package skills

import (
	"net"
	"testing"
)

// TestDenyByDefault_S2 confirms the PolicyEnforcer denies unknown skills.
// Any skill name not present in the Policy map must return false.
func TestDenyByDefault_S2(t *testing.T) {
	pe := NewPolicyEnforcer()
	if pe.IsAllowed("unknown.nonexistent.skill") {
		t.Fatal("expected IsAllowed to return false for unknown skill, got true")
	}
}

// TestYAMLv3Parser_S3 confirms parseYAMLFrontmatter uses yaml.v3 and
// correctly parses nested objects without errors.
func TestYAMLv3Parser_S3(t *testing.T) {
	input := `name: test-skill
metadata:
  openclaw:
    version: "1.0"
`
	var result map[string]interface{}
	if err := parseYAMLFrontmatter(input, &result); err != nil {
		t.Fatalf("parseYAMLFrontmatter returned error: %v", err)
	}

	name, ok := result["name"].(string)
	if !ok || name != "test-skill" {
		t.Fatalf("expected name='test-skill', got %v", result["name"])
	}

	meta, ok := result["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("expected metadata to be a map")
	}
	openclaw, ok := meta["openclaw"].(map[string]interface{})
	if !ok {
		t.Fatal("expected metadata.openclaw to be a map")
	}
	ver, ok := openclaw["version"].(string)
	if !ok || ver != "1.0" {
		t.Fatalf("expected metadata.openclaw.version='1.0', got %v", openclaw["version"])
	}
}

// TestIPv6Blocking_S6 confirms that IPv6 loopback, unique-local, link-local,
// and IPv4-mapped addresses are all detected as private.
func TestIPv6Blocking_S6(t *testing.T) {
	v := NewSSRFValidator()
	cases := []struct {
		name string
		ip   string
	}{
		{"IPv6 loopback", "::1"},
		{"IPv6 unique-local", "fc00::1"},
		{"IPv6 link-local", "fe80::1"},
		{"IPv4-mapped IPv6", "::ffff:127.0.0.1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %s", tc.ip)
			}
			if !v.isPrivateIP(ip) {
				t.Errorf("expected %s (%s) to be private, but isPrivateIP returned false", tc.name, tc.ip)
			}
		})
	}
}

// TestExtractHostStripsUserinfo_S7 confirms that extractHost strips userinfo
// from URLs, preventing SSRF bypass via userinfo@host patterns.
func TestExtractHostStripsUserinfo_S7(t *testing.T) {
	v := NewSSRFValidator()
	host := v.extractHost("https://attacker@169.254.169.254/metadata")
	if host != "169.254.169.254" {
		t.Fatalf("expected host='169.254.169.254', got '%s'", host)
	}
	// Further verify the host is recognized as a metadata endpoint
	if !v.isMetadataEndpoint(host) {
		t.Error("expected 169.254.169.254 to be a metadata endpoint")
	}
}

// TestEmailSendPolicy_S8 confirms that email.send has high risk and
// requires explicit approval (AutoExecute == false).
func TestEmailSendPolicy_S8(t *testing.T) {
	policy, exists := Policy["email.send"]
	if !exists {
		t.Fatal("Policy map missing 'email.send' entry")
	}
	if policy.Risk != "high" {
		t.Errorf("expected email.send Risk='high', got '%s'", policy.Risk)
	}
	if policy.AutoExecute {
		t.Error("expected email.send AutoExecute=false, got true")
	}
}

// TestDefaultGovernorInstalled_F1 confirms that NewSkillExecutor installs
// a non-nil governor (skillGate) by default.
func TestDefaultGovernorInstalled_F1(t *testing.T) {
	se := NewSkillExecutor()
	if se.skillGate == nil {
		t.Fatal("expected skillGate to be non-nil after NewSkillExecutor()")
	}
}

use crate::error::{Result, SidecarError};
use image::{DynamicImage, ImageFormat, ImageReader};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::io::Cursor;
use std::path::Path;
use std::process::Command;
use tract_onnx::prelude::*;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OcrResult {
    pub text: String,
    pub confidence: f32,
    pub language: String,
    pub metadata: HashMap<String, String>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
pub enum OcrBackend {
    #[serde(rename = "tesseract")]
    Tesseract,
    #[serde(rename = "onnx")]
    Onnx,
    #[serde(rename = "auto")]
    #[default]
    Auto,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OcrConfig {
    pub language: String,
    pub dpi: Option<u32>,
    pub psm: Option<u32>,
    #[serde(default)]
    pub backend: OcrBackend,
    /// Enable ONNX fallback when Tesseract fails or is unavailable (default: true).
    #[serde(default = "default_true")]
    pub onnx_fallback: bool,
    /// Path to the ONNX model file used for fallback OCR inference.
    /// When `None`, a default path of `/opt/armorclaw/models/ocr.onnx` is used.
    #[serde(default)]
    pub onnx_model_path: Option<String>,
}

fn default_true() -> bool {
    true
}

impl Default for OcrConfig {
    fn default() -> Self {
        Self {
            language: "eng".to_string(),
            dpi: Some(300),
            psm: Some(3),
            backend: OcrBackend::Auto,
            onnx_fallback: true,
            onnx_model_path: None,
        }
    }
}

pub struct OcrExtractor {
    config: OcrConfig,
}

impl OcrExtractor {
    pub fn new(config: OcrConfig) -> Self {
        Self { config }
    }

    pub fn extract(&self, image_data: &[u8]) -> Result<OcrResult> {
        let format = detect_image_format(image_data)?;

        let img = ImageReader::with_format(Cursor::new(image_data), format)
            .decode()
            .map_err(|e| {
                SidecarError::DocumentProcessingError(format!("Failed to decode image: {}", e))
            })?;

        match self.config.backend {
            OcrBackend::Tesseract => self.extract_tesseract_primary(&img, image_data),
            OcrBackend::Onnx => self.extract_onnx_only(image_data),
            OcrBackend::Auto => self.extract_auto(&img, image_data),
        }
    }

    fn extract_auto(&self, img: &DynamicImage, image_data: &[u8]) -> Result<OcrResult> {
        let tesseract_available = self.check_tesseract_installed();

        if tesseract_available {
            match self.run_tesseract(img) {
                Ok(result) if !result.text.trim().is_empty() => return Ok(result),
                Ok(_) => {
                    // Tesseract returned empty text — try ONNX fallback
                }
                Err(_) => {
                    // Tesseract failed — try ONNX fallback
                }
            }
        }

        if self.config.onnx_fallback {
            return self.run_onnx_fallback(image_data);
        }

        if !tesseract_available {
            return Err(SidecarError::InvalidRequest(
                "Tesseract OCR is not installed and ONNX fallback is disabled. Install tesseract-ocr or enable ONNX fallback.".to_string()
            ));
        }

        Err(SidecarError::DocumentProcessingError(
            "Tesseract returned no text and ONNX fallback is disabled.".to_string(),
        ))
    }

    fn extract_tesseract_primary(
        &self,
        img: &DynamicImage,
        image_data: &[u8],
    ) -> Result<OcrResult> {
        let tesseract_available = self.check_tesseract_installed();

        if !tesseract_available {
            if self.config.onnx_fallback {
                return self.run_onnx_fallback(image_data);
            }
            return Err(SidecarError::InvalidRequest(
                "Tesseract OCR is not installed. Install with: apt-get install tesseract-ocr (Debian/Ubuntu) or brew install tesseract (macOS)".to_string()
            ));
        }

        let result = self.run_tesseract(img)?;
        if result.text.trim().is_empty() && self.config.onnx_fallback {
            return self.run_onnx_fallback(image_data);
        }
        Ok(result)
    }

    fn extract_onnx_only(&self, image_data: &[u8]) -> Result<OcrResult> {
        self.run_onnx_fallback(image_data)
    }

    fn run_onnx_fallback(&self, image_data: &[u8]) -> Result<OcrResult> {
        let model_path = self
            .config
            .onnx_model_path
            .as_deref()
            .unwrap_or("/opt/armorclaw/models/ocr.onnx");

        let backend = OnnxBackend::new(model_path)?;
        backend.extract(image_data)
    }

    fn check_tesseract_installed(&self) -> bool {
        Command::new("tesseract").arg("--version").output().is_ok()
    }

    fn run_tesseract(&self, img: &DynamicImage) -> Result<OcrResult> {
        let temp_input = tempfile::NamedTempFile::with_suffix(".png").map_err(|e| {
            SidecarError::DocumentProcessingError(format!("Failed to create temp file: {}", e))
        })?;

        let input_path = temp_input.path().to_path_buf();

        img.save(&input_path).map_err(|e| {
            SidecarError::DocumentProcessingError(format!("Failed to save image: {}", e))
        })?;

        let output_base = input_path.with_extension("");

        let mut cmd = Command::new("tesseract");
        cmd.arg(input_path);
        cmd.arg(&output_base);
        cmd.arg("-l");
        cmd.arg(&self.config.language);
        cmd.arg("--psm");
        cmd.arg(self.config.psm.unwrap_or(3).to_string());

        let output = cmd.output().map_err(|e| {
            SidecarError::DocumentProcessingError(format!("Failed to run tesseract: {}", e))
        })?;

        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            return Err(SidecarError::DocumentProcessingError(format!(
                "Tesseract failed: {}",
                stderr
            )));
        }

        let output_file = format!("{}.txt", output_base.display());
        let text = std::fs::read_to_string(&output_file).map_err(|e| {
            SidecarError::DocumentProcessingError(format!("Failed to read OCR output: {}", e))
        })?;

        let _ = std::fs::remove_file(&output_file);

        let confidence = self.estimate_confidence(&text);

        let mut metadata = HashMap::new();
        metadata.insert("engine".to_string(), "tesseract".to_string());
        metadata.insert("language".to_string(), self.config.language.clone());
        metadata.insert("psm".to_string(), self.config.psm.unwrap_or(3).to_string());

        Ok(OcrResult {
            text: text.trim().to_string(),
            confidence,
            language: self.config.language.clone(),
            metadata,
        })
    }

    fn estimate_confidence(&self, text: &str) -> f32 {
        if text.is_empty() {
            return 0.0;
        }

        let words: Vec<&str> = text.split_whitespace().collect();
        if words.is_empty() {
            return 0.0;
        }

        let valid_words: Vec<&str> = words
            .iter()
            .filter(|w| {
                w.chars()
                    .all(|c| c.is_alphanumeric() || c.is_whitespace() || "-'".contains(c))
            })
            .copied()
            .collect();

        let ratio = valid_words.len() as f32 / words.len() as f32;
        0.5 + (ratio * 0.4)
    }
}

impl Default for OcrExtractor {
    fn default() -> Self {
        Self::new(OcrConfig::default())
    }
}

pub fn extract_text_with_ocr(image_data: &[u8], config: Option<OcrConfig>) -> Result<OcrResult> {
    let extractor = OcrExtractor::new(config.unwrap_or_default());
    extractor.extract(image_data)
}

pub fn validate_language(lang: &str) -> Result<()> {
    let supported = get_supported_languages();
    if supported.contains(&lang.to_string()) {
        Ok(())
    } else {
        Err(SidecarError::InvalidRequest(format!(
            "Unsupported language: {}. Supported: {:?}",
            lang, supported
        )))
    }
}

pub fn get_supported_languages() -> Vec<String> {
    vec![
        "eng".to_string(),
        "spa".to_string(),
        "fra".to_string(),
        "deu".to_string(),
        "ita".to_string(),
        "por".to_string(),
        "rus".to_string(),
        "chi_sim".to_string(),
        "chi_tra".to_string(),
        "jpn".to_string(),
        "kor".to_string(),
        "ara".to_string(),
        "hin".to_string(),
        "nld".to_string(),
        "pol".to_string(),
        "tur".to_string(),
    ]
}

pub fn detect_language_from_text(text: &str) -> String {
    if text.is_empty() {
        return "eng".to_string();
    }

    let cyrillic = text
        .chars()
        .filter(|c| (*c >= 'а' && *c <= 'я') || (*c >= 'А' && *c <= 'Я'))
        .count();
    let cjk = text
        .chars()
        .filter(|c| (*c >= '\u{4E00}' && *c <= '\u{9FFF}'))
        .count();
    let latin = text.chars().filter(|c| c.is_ascii_alphabetic()).count();

    let total = cyrillic + cjk + latin;
    if total == 0 {
        return "eng".to_string();
    }

    if cyrillic as f32 / total as f32 > 0.5 {
        return "rus".to_string();
    }
    if cjk as f32 / total as f32 > 0.5 {
        return "chi_sim".to_string();
    }
    "eng".to_string()
}

const ONNX_FALLBACK_THRESHOLD: usize = 50;

pub struct OnnxBackend {
    model: RunnableModel<TypedFact, Box<dyn TypedOp>, Graph<TypedFact, Box<dyn TypedOp>>>,
    model_path: String,
}

impl OnnxBackend {
    pub fn new(model_path: &str) -> Result<Self> {
        if !Path::new(model_path).exists() {
            return Err(SidecarError::DocumentProcessingError(format!(
                "ONNX model not found at: {}",
                model_path
            )));
        }

        let model = tract_onnx::onnx()
            .model_for_path(model_path)
            .map_err(|e| {
                SidecarError::DocumentProcessingError(format!("Failed to load ONNX model: {}", e))
            })?
            .into_optimized()
            .map_err(|e| {
                SidecarError::DocumentProcessingError(format!(
                    "Failed to optimize ONNX model: {}",
                    e
                ))
            })?
            .into_runnable()
            .map_err(|e| {
                SidecarError::DocumentProcessingError(format!(
                    "Failed to make ONNX model runnable: {}",
                    e
                ))
            })?;

        Ok(Self {
            model,
            model_path: model_path.to_string(),
        })
    }

    pub fn model_path(&self) -> &str {
        &self.model_path
    }

    pub fn extract(&self, image_data: &[u8]) -> Result<OcrResult> {
        let img = ImageReader::with_format(Cursor::new(image_data), ImageFormat::Png)
            .decode()
            .map_err(|e| {
                SidecarError::DocumentProcessingError(format!(
                    "Failed to decode image for ONNX: {}",
                    e
                ))
            })?;

        let rgb = img.to_rgb8();
        let resized =
            image::imageops::resize(&rgb, 320, 320, image::imageops::FilterType::Triangle);

        let tensor: Tensor =
            tract_ndarray::Array4::from_shape_fn((1, 3, 320, 320), |(_, c, y, x)| {
                resized[(x as _, y as _)][c] as f32 / 255.0
            })
            .into();

        let result = self.model.run(tvec!(tensor.into())).map_err(|e| {
            SidecarError::DocumentProcessingError(format!("ONNX inference failed: {}", e))
        })?;

        let text = extract_text_from_onnx_output(&result);

        let confidence = estimate_confidence_from_text(&text);

        let mut metadata = HashMap::new();
        metadata.insert("engine".to_string(), "onnx".to_string());
        metadata.insert("model_path".to_string(), self.model_path.clone());

        Ok(OcrResult {
            text,
            confidence,
            language: "auto".to_string(),
            metadata,
        })
    }
}

fn extract_text_from_onnx_output(outputs: &[TValue]) -> String {
    if outputs.is_empty() {
        return String::new();
    }

    match outputs[0].to_array_view::<f32>() {
        Ok(view) => {
            let mut chars = Vec::new();
            for &val in view.iter() {
                let idx = val.round() as usize;
                if idx == 0 {
                    break;
                }
                if let Some(c) = char::from_u32(idx as u32 + 0x20) {
                    if c.is_ascii_graphic() || c.is_ascii_whitespace() {
                        chars.push(c);
                    }
                }
            }
            chars.into_iter().collect()
        }
        Err(_) => String::new(),
    }
}

fn estimate_confidence_from_text(text: &str) -> f32 {
    if text.is_empty() {
        return 0.0;
    }
    let words: Vec<&str> = text.split_whitespace().collect();
    if words.is_empty() {
        return 0.0;
    }
    let valid: Vec<&str> = words
        .iter()
        .filter(|w| {
            w.chars()
                .all(|c| c.is_alphanumeric() || c.is_whitespace() || "-'".contains(c))
        })
        .copied()
        .collect();
    0.5 + (valid.len() as f32 / words.len() as f32 * 0.4)
}

pub fn should_fallback_to_tesseract(onnx_text: &str, tesseract_available: bool) -> bool {
    if !tesseract_available {
        return false;
    }
    onnx_text.len() < ONNX_FALLBACK_THRESHOLD
}

fn detect_image_format(data: &[u8]) -> Result<ImageFormat> {
    if data.len() < 8 {
        return Err(SidecarError::DocumentProcessingError(
            "Image data too small".to_string(),
        ));
    }

    if data[0..8] == [137, 80, 78, 71, 13, 10, 26, 10] {
        return Ok(ImageFormat::Png);
    }

    if data[0..2] == [0xFF, 0xD8] {
        return Ok(ImageFormat::Jpeg);
    }

    if data.len() >= 12 && &data[0..4] == b"RIFF" && &data[8..12] == b"WEBP" {
        return Ok(ImageFormat::WebP);
    }

    if data[0..6] == [71, 73, 70, 56, 57, 97] || data[0..6] == [71, 73, 70, 56, 55, 97] {
        return Ok(ImageFormat::Gif);
    }

    if &data[0..2] == b"BM" {
        return Ok(ImageFormat::Bmp);
    }

    if &data[0..4] == b"II*\x00" || &data[0..4] == b"MM\x00*" {
        return Ok(ImageFormat::Tiff);
    }

    Ok(ImageFormat::Png)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ocr_extractor_creation() {
        let extractor = OcrExtractor::default();
        assert_eq!(extractor.config.language, "eng");
    }

    #[test]
    fn test_ocr_config_default() {
        let config = OcrConfig::default();
        assert_eq!(config.language, "eng");
        assert_eq!(config.dpi, Some(300));
        assert_eq!(config.psm, Some(3));
        assert!(config.onnx_fallback);
        assert!(config.onnx_model_path.is_none());
    }

    #[test]
    fn test_validate_language() {
        assert!(validate_language("eng").is_ok());
        assert!(validate_language("spa").is_ok());
        assert!(validate_language("invalid").is_err());
    }

    #[test]
    fn test_supported_languages() {
        let langs = get_supported_languages();
        assert!(langs.contains(&"eng".to_string()));
        assert!(langs.contains(&"spa".to_string()));
        assert!(langs.contains(&"chi_sim".to_string()));
        assert!(langs.len() >= 10);
    }

    #[test]
    fn test_detect_language_from_text_empty() {
        let result = detect_language_from_text("");
        assert_eq!(result, "eng");
    }

    #[test]
    fn test_detect_language_from_text_english() {
        let result = detect_language_from_text("Hello world this is English text");
        assert_eq!(result, "eng");
    }

    #[test]
    fn test_detect_language_from_text_russian() {
        let result = detect_language_from_text("Привет мир это русский текст");
        assert_eq!(result, "rus");
    }

    #[test]
    fn test_detect_language_from_text_chinese() {
        let result = detect_language_from_text("你好世界这是中文文本");
        assert_eq!(result, "chi_sim");
    }

    #[test]
    fn test_estimate_confidence_empty() {
        let extractor = OcrExtractor::default();
        let result = extractor.estimate_confidence("");
        assert_eq!(result, 0.0);
    }

    #[test]
    fn test_estimate_confidence_valid() {
        let extractor = OcrExtractor::default();
        let result = extractor.estimate_confidence("Hello world this is valid text");
        assert!(result > 0.8);
    }

    #[test]
    fn test_detect_image_format_png() {
        let png_header = vec![137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 0];
        let result = detect_image_format(&png_header);
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), ImageFormat::Png);
    }

    #[test]
    fn test_detect_image_format_jpeg() {
        let jpeg_header = vec![0xFF, 0xD8, 0xFF, 0xE0, 0, 0, 0, 0];
        let result = detect_image_format(&jpeg_header);
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), ImageFormat::Jpeg);
    }

    #[test]
    fn test_detect_image_format_too_small() {
        let small_data = vec![0, 1, 2];
        let result = detect_image_format(&small_data);
        assert!(result.is_err());
    }

    #[test]
    fn test_ocr_backend_serialization() {
        let tesseract = OcrBackend::Tesseract;
        let onnx = OcrBackend::Onnx;
        let auto = OcrBackend::Auto;

        assert_eq!(serde_json::to_string(&tesseract).unwrap(), r#""tesseract""#);
        assert_eq!(serde_json::to_string(&onnx).unwrap(), r#""onnx""#);
        assert_eq!(serde_json::to_string(&auto).unwrap(), r#""auto""#);
    }

    #[test]
    fn test_ocr_backend_deserialization() {
        let tesseract: OcrBackend = serde_json::from_str(r#""tesseract""#).unwrap();
        let onnx: OcrBackend = serde_json::from_str(r#""onnx""#).unwrap();
        let auto: OcrBackend = serde_json::from_str(r#""auto""#).unwrap();

        assert!(matches!(tesseract, OcrBackend::Tesseract));
        assert!(matches!(onnx, OcrBackend::Onnx));
        assert!(matches!(auto, OcrBackend::Auto));
    }

    #[test]
    fn test_ocr_config_with_backend() {
        let config = OcrConfig {
            language: "eng".to_string(),
            dpi: Some(300),
            psm: Some(3),
            backend: OcrBackend::Onnx,
            onnx_fallback: true,
            onnx_model_path: None,
        };
        assert!(matches!(config.backend, OcrBackend::Onnx));

        let default_config = OcrConfig::default();
        assert!(matches!(default_config.backend, OcrBackend::Auto));
    }

    #[test]
    fn test_onnx_backend_missing_model_returns_error() {
        let result = OnnxBackend::new("/nonexistent/path/model.onnx");
        assert!(result.is_err());
    }

    #[test]
    fn test_onnx_backend_extract_missing_model() {
        let backend = OnnxBackend::new("/nonexistent/path/model.onnx");
        assert!(backend.is_err());
    }

    #[test]
    fn test_auto_fallback_threshold_short_text() {
        let short_text = "hi";
        assert!(should_fallback_to_tesseract(short_text, true));
    }

    #[test]
    fn test_auto_fallback_threshold_long_text() {
        let long_text =
            "This is a sufficiently long piece of extracted text that should not trigger fallback.";
        assert!(!should_fallback_to_tesseract(long_text, true));
    }

    #[test]
    fn test_auto_fallback_tesseract_unavailable() {
        let short_text = "hi";
        assert!(!should_fallback_to_tesseract(short_text, false));
    }

    #[test]
    fn test_confidence_estimation_with_metadata() {
        let extractor = OcrExtractor::default();
        let result = extractor.estimate_confidence("hello world");
        assert!(result > 0.5);
        assert!(result <= 1.0);
    }

    #[test]
    fn test_onnx_fallback_disabled_config() {
        let config = OcrConfig {
            onnx_fallback: false,
            ..OcrConfig::default()
        };
        assert!(!config.onnx_fallback);
    }

    #[test]
    fn test_onnx_fallback_default_enabled() {
        let config = OcrConfig::default();
        assert!(config.onnx_fallback);
    }

    #[test]
    fn test_extract_onnx_backend_missing_model_returns_error() {
        let config = OcrConfig {
            backend: OcrBackend::Onnx,
            onnx_model_path: Some("/nonexistent/path/model.onnx".to_string()),
            ..OcrConfig::default()
        };
        let extractor = OcrExtractor::new(config);

        let png_header = vec![137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 0];
        let result = extractor.extract(&png_header);
        assert!(result.is_err());
    }

    #[test]
    fn test_tesseract_primary_falls_back_to_onnx_on_missing_model() {
        let config = OcrConfig {
            backend: OcrBackend::Tesseract,
            onnx_fallback: true,
            onnx_model_path: Some("/nonexistent/path/model.onnx".to_string()),
            ..OcrConfig::default()
        };
        let extractor = OcrExtractor::new(config);

        let png_header = vec![137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 0];
        let result = extractor.extract(&png_header);

        if !extractor.check_tesseract_installed() {
            assert!(result.is_err());
        }
    }

    #[test]
    fn test_auto_mode_no_tesseract_no_onnx_model_returns_error() {
        let config = OcrConfig {
            backend: OcrBackend::Auto,
            onnx_fallback: true,
            onnx_model_path: Some("/nonexistent/path/model.onnx".to_string()),
            ..OcrConfig::default()
        };
        let extractor = OcrExtractor::new(config);

        let png_header = vec![137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 0];
        let result = extractor.extract(&png_header);

        if !extractor.check_tesseract_installed() {
            assert!(result.is_err());
        }
    }

    #[test]
    fn test_onnx_fallback_disabled_no_tesseract_returns_error() {
        let config = OcrConfig {
            backend: OcrBackend::Auto,
            onnx_fallback: false,
            ..OcrConfig::default()
        };
        let extractor = OcrExtractor::new(config);

        if !extractor.check_tesseract_installed() {
            let png_header = vec![137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 0];
            let result = extractor.extract(&png_header);
            assert!(result.is_err());
        }
    }

    #[test]
    fn test_extract_text_from_onnx_output_empty() {
        let result = extract_text_from_onnx_output(&[]);
        assert!(result.is_empty());
    }
}

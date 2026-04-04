use crate::error::SidecarError;
use crate::document::diff::{DiffResult, DiffOp, DiffEngine};

pub struct HtmlDiffGenerator;

impl HtmlDiffGenerator {
    pub fn new() -> Self {
        Self
    }

    pub fn generate_html_diff(
        &self,
        old_text: &str,
        new_text: &str,
    ) -> Result<String, SidecarError> {
        let diff = DiffEngine::new().compute_diff(old_text, new_text);
        let mut html = String::new();

        html.push_str("<!DOCTYPE html>\n");
        html.push_str("<html lang=\"en\">\n");
        html.push_str("<head>\n");
        html.push_str("<meta charset=\"UTF-8\">\n");
        html.push_str("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n");
        html.push_str("<title>Document Diff</title>\n");
        html.push_str("<style>\n");
        html.push_str("body { font-family: monospace; margin: 20px; }\n");
        html.push_str(".insertion { background-color: #d4edda; color: #155724; }\n");
        html.push_str(".deletion { background-color: #f8d7da; color: #721c24; }\n");
        html.push_str(".equal { background-color: #f8f9fa; color: #383d41; }\n");
        html.push_str(".line-number { color: #6c757d; min-width: 50px; display: inline-block; }\n");
        html.push_str(".line { margin: 2px 0; padding: 2px 5px; }\n");
        html.push_str("</style>\n");
        html.push_str("</head>\n");
        html.push_str("<body>\n");

        let mut line_num = 1;

        for insertion in &diff.insertions {
            if let DiffOp::Insert(text) = insertion {
                html.push_str(&format!(
                    "<div class=\"line\"><span class=\"line-number\">{}</span><span class=\"insertion\">+{}</span></div>\n",
                    line_num,
                    Self::escape_html(text)
                ));
                line_num += 1;
            }
        }

        for deletion in &diff.deletions {
            if let DiffOp::Delete(text) = deletion {
                html.push_str(&format!(
                    "<div class=\"line\"><span class=\"line-number\">{}</span><span class=\"deletion\">-{}</span></div>\n",
                    line_num,
                    Self::escape_html(text)
                ));
                line_num += 1;
            }
        }

        for (idx, text) in &diff.equals {
            html.push_str(&format!(
                "<div class=\"line\"><span class=\"line-number\">{}</span><span class=\"equal\">{}</span></div>\n",
                idx + 1,
                Self::escape_html(text)
            ));
        }

        html.push_str("</body>\n");
        html.push_str("</html>\n");

        Ok(html)
    }

    fn escape_html(text: &str) -> String {
        text.replace('&', "&amp;")
            .replace('<', "&lt;")
            .replace('>', "&gt;")
            .replace('"', "&quot;")
            .replace('\'', "&#39;")
    }
}

pub fn generate_html_diff(old_text: &str, new_text: &str) -> Result<String, SidecarError> {
    HtmlDiffGenerator::new().generate_html_diff(old_text, new_text)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_html_diff_generator_new() {
        let _generator = HtmlDiffGenerator::new();
    }

    #[test]
    fn test_empty_texts() {
        let result = HtmlDiffGenerator::new().generate_html_diff("", "");
        assert!(result.is_ok());
        let html = result.unwrap();
        assert!(html.contains("<!DOCTYPE html>"));
        assert!(html.contains("</html>"));
    }

    #[test]
    fn test_identical_texts() {
        let text = "Hello\nWorld";
        let result = HtmlDiffGenerator::new().generate_html_diff(text, text);
        assert!(result.is_ok());
        let html = result.unwrap();
        assert!(html.contains("equal"));
    }

    #[test]
    fn test_simple_insertion() {
        let old = "Hello\nWorld";
        let new = "Hello\nWorld\nNew Line";
        let result = HtmlDiffGenerator::new().generate_html_diff(old, new);
        assert!(result.is_ok());
        let html = result.unwrap();
        assert!(html.contains("insertion"));
        assert!(html.contains("New Line"));
    }

    #[test]
    fn test_simple_deletion() {
        let old = "Hello\nWorld\nOld Line";
        let new = "Hello\nWorld";
        let result = HtmlDiffGenerator::new().generate_html_diff(old, new);
        assert!(result.is_ok());
        let html = result.unwrap();
        assert!(html.contains("deletion"));
        assert!(html.contains("Old Line"));
    }

    #[test]
    fn test_html_escaping() {
        let old = "Test <script>";
        let new = "Test <b>bold</b>";
        let result = HtmlDiffGenerator::new().generate_html_diff(old, new);
        assert!(result.is_ok());
        let html = result.unwrap();
        assert!(html.contains("&lt;script&gt;"));
        assert!(html.contains("&lt;b&gt;bold&lt;/b&gt;"));
    }

    #[test]
    fn test_complete_html_structure() {
        let result = HtmlDiffGenerator::new().generate_html_diff("Old", "New");
        assert!(result.is_ok());
        let html = result.unwrap();
        assert!(html.contains("<!DOCTYPE html>"));
        assert!(html.contains("<html"));
        assert!(html.contains("<head>"));
        assert!(html.contains("<body>"));
        assert!(html.contains("</body>"));
        assert!(html.contains("</html>"));
    }

    #[test]
    fn test_css_styling() {
        let result = HtmlDiffGenerator::new().generate_html_diff("Test", "Test2");
        assert!(result.is_ok());
        let html = result.unwrap();
        assert!(html.contains("<style>"));
        assert!(html.contains(".insertion"));
        assert!(html.contains(".deletion"));
        assert!(html.contains(".equal"));
    }

    #[test]
    fn test_line_numbers() {
        let result = HtmlDiffGenerator::new().generate_html_diff("Line 1\nLine 2", "Line 1\nModified");
        assert!(result.is_ok());
        let html = result.unwrap();
        assert!(html.contains("line-number"));
    }
}

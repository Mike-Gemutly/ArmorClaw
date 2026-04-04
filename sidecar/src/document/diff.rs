use crate::error::SidecarError;
use std::collections::HashMap;

use crate::document::validate_file_size;

pub struct DiffEngine;

#[derive(Debug, Clone, PartialEq)]
pub enum DiffOp {
    Insert(String),
    Delete(String),
    Equal,
}

#[derive(Debug, Clone)]
pub struct DiffResult {
    pub insertions: Vec<DiffOp>,
    pub deletions: Vec<DiffOp>,
    pub equals: Vec<(usize, String)>,
}

impl DiffEngine {
    pub fn new() -> Self {
        Self
    }

    pub fn compute_diff(&self, old_text: &str, new_text: &str) -> DiffResult {
        let mut insertions = Vec::new();
        let mut deletions = Vec::new();
        let mut equals = Vec::new();

        let old_lines: Vec<&str> = old_text.lines().collect();
        let new_lines: Vec<&str> = new_text.lines().collect();

        let mut old_idx = 0;
        let mut new_idx = 0;

        while old_idx < old_lines.len() || new_idx < new_lines.len() {
            if old_idx >= old_lines.len() {
                for line in &new_lines[new_idx..] {
                    insertions.push(DiffOp::Insert(line.to_string()));
                }
                break;
            }

            if new_idx >= new_lines.len() {
                for line in &old_lines[old_idx..] {
                    deletions.push(DiffOp::Delete(line.to_string()));
                }
                break;
            }

            let old_line = old_lines[old_idx];
            let new_line = new_lines[new_idx];

            if old_line == new_line {
                equals.push((old_idx, old_line.to_string()));
                old_idx += 1;
                new_idx += 1;
            } else {
                deletions.push(DiffOp::Delete(old_line.to_string()));
                insertions.push(DiffOp::Insert(new_line.to_string()));
                old_idx += 1;
                new_idx += 1;
            }
        }

        DiffResult {
            insertions,
            deletions,
            equals,
        }
    }
}

pub fn compute_text_diff(old: &str, new: &str) -> DiffResult {
    DiffEngine::new().compute_diff(old, new)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_diff_engine_new() {
        let engine = DiffEngine::new();
        assert!(engine.compute_diff("", "").equals.is_empty());
    }

    #[test]
    fn test_empty_strings() {
        let result = DiffEngine::new().compute_diff("", "");
        assert!(result.insertions.is_empty());
        assert!(result.deletions.is_empty());
        assert!(result.equals.is_empty());
    }

    #[test]
    fn test_identical_texts() {
        let text = "Hello\nWorld\nTest";
        let result = DiffEngine::new().compute_diff(text, text);
        assert!(result.insertions.is_empty());
        assert!(result.deletions.is_empty());
        assert_eq!(result.equals.len(), 3);
    }

    #[test]
    fn test_simple_insertion() {
        let old = "Hello\nWorld";
        let new = "Hello\nWorld\nNew Line";
        let result = DiffEngine::new().compute_diff(old, new);
        assert_eq!(result.insertions.len(), 1);
        assert!(matches!(&result.insertions[0], DiffOp::Insert(_)));
    }

    #[test]
    fn test_simple_deletion() {
        let old = "Hello\nWorld\nOld Line";
        let new = "Hello\nWorld";
        let result = DiffEngine::new().compute_diff(old, new);
        assert_eq!(result.deletions.len(), 1);
        assert!(matches!(&result.deletions[0], DiffOp::Delete(_)));
    }

    #[test]
    fn test_line_replacement() {
        let old = "Hello\nWorld";
        let new = "Hello\nUniverse";
        let result = DiffEngine::new().compute_diff(old, new);
        assert_eq!(result.deletions.len(), 1);
        assert_eq!(result.insertions.len(), 1);
    }

    #[test]
    fn test_complex_diff() {
        let old = "Line 1\nLine 2\nLine 3";
        let new = "Line 1\nModified\nLine 3\nNew";
        let result = DiffEngine::new().compute_diff(old, new);
        assert!(result.insertions.len() >= 2);
        assert!(result.deletions.len() >= 1);
        assert!(result.equals.len() >= 1);
    }

    #[test]
    fn test_large_text() {
        let old: String = (1..100).map(|i| format!("Line {}", i)).collect::<Vec<String>>().join("\n");
        let new: String = (1..110).map(|i| format!("Line {}", i)).collect::<Vec<String>>().join("\n");
        let result = DiffEngine::new().compute_diff(&old, &new);
        assert!(result.insertions.len() > 0);
        assert!(result.deletions.is_empty());
        assert!(result.equals.len() > 90);
    }

    #[test]
    fn test_unicode_support() {
        let old = "Hello\n世界\nTest";
        let new = "Hello\nUniverse\nTest";
        let result = DiffEngine::new().compute_diff(old, new);
        assert_eq!(result.deletions.len(), 1);
        assert_eq!(result.insertions.len(), 1);
    }

    #[test]
    fn test_empty_vs_content() {
        let result = DiffEngine::new().compute_diff("", "New content");
        assert_eq!(result.insertions.len(), 1);
        assert!(result.deletions.is_empty());
        assert!(result.equals.is_empty());
    }
}

use crate::error::{Result, SidecarError};
use crate::security::shadowmap::ShadowMap;
use calamine::{open_workbook_from_rs, Reader, Xlsx};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::io::Cursor;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct XlsxExtractionResult {
    pub sheets: Vec<SheetData>,
    pub metadata: HashMap<String, String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SheetData {
    pub name: String,
    pub rows: Vec<Vec<Option<String>>>,
    pub formulas: Vec<FormulaCell>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FormulaCell {
    pub row: usize,
    pub col: usize,
    pub formula: String,
    pub value: Option<String>,
}

pub struct XlsxExtractor;

impl XlsxExtractor {
    pub fn new() -> Self {
        Self
    }

    pub fn extract(&self, data: &[u8]) -> Result<XlsxExtractionResult> {
        let cursor = Cursor::new(data);
        let mut workbook: Xlsx<_> = open_workbook_from_rs(cursor).map_err(|e| {
            SidecarError::DocumentProcessingError(format!("Failed to open XLSX: {}", e))
        })?;

        let mut sheets = Vec::new();
        let mut metadata = HashMap::new();

        let sheet_names = workbook.sheet_names();
        metadata.insert("sheet_count".to_string(), sheet_names.len().to_string());

        for sheet_name in sheet_names {
            let range = workbook.worksheet_range(&sheet_name).map_err(|e| {
                SidecarError::DocumentProcessingError(format!(
                    "Failed to read sheet '{}': {}",
                    sheet_name, e
                ))
            })?;

            let mut rows = Vec::new();
            let height = range.height();
            let width = range.width();

            for row_idx in 0..height {
                let mut row = Vec::new();
                for col_idx in 0..width {
                    let cell_value = range.get_value((row_idx as u32, col_idx as u32));
                    let value_str = cell_value.map(|v| format_cell_value(v.clone()));
                    row.push(value_str);
                }
                rows.push(row);
            }

            sheets.push(SheetData {
                name: sheet_name.clone(),
                rows,
                formulas: Vec::new(),
            });
        }

        Ok(XlsxExtractionResult { sheets, metadata })
    }

    pub fn extract_formulas(&self, _data: &[u8]) -> Result<Vec<FormulaCell>> {
        Ok(Vec::new())
    }

    pub fn extract_redacted(
        &self,
        data: &[u8],
        shadowmap: &mut ShadowMap,
    ) -> Result<XlsxExtractionResult> {
        let mut result = self.extract(data)?;
        for sheet in &mut result.sheets {
            for row in &mut sheet.rows {
                for cell in row.iter_mut() {
                    if let Some(ref mut value) = cell {
                        *value = shadowmap.redact(value);
                    }
                }
            }
        }
        Ok(result)
    }
}

impl Default for XlsxExtractor {
    fn default() -> Self {
        Self::new()
    }
}

pub fn extract_data_from_xlsx(data: &[u8]) -> Result<XlsxExtractionResult> {
    XlsxExtractor::new().extract(data)
}

fn format_cell_value(value: calamine::Data) -> String {
    match value {
        calamine::Data::Empty => String::new(),
        calamine::Data::String(s) => s,
        calamine::Data::Float(f) => {
            if f.fract() == 0.0 {
                format!("{}", f as i64)
            } else {
                format!("{}", f)
            }
        }
        calamine::Data::Int(i) => format!("{}", i),
        calamine::Data::Bool(b) => format!("{}", b),
        calamine::Data::DateTime(dt) => format!("{}", dt),
        calamine::Data::Error(e) => format!("#ERROR: {:?}", e),
        calamine::Data::DateTimeIso(dt) => dt.to_string(),
        calamine::Data::DurationIso(d) => d.to_string(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_xlsx_extractor_creation() {
        let extractor = XlsxExtractor::new();
        assert!(extractor.extract(&[]).is_err());
    }

    #[test]
    fn test_format_cell_value_empty() {
        let result = format_cell_value(calamine::Data::Empty);
        assert!(result.is_empty());
    }

    #[test]
    fn test_format_cell_value_string() {
        let result = format_cell_value(calamine::Data::String("test".to_string()));
        assert_eq!(result, "test");
    }

    #[test]
    fn test_format_cell_value_int() {
        let result = format_cell_value(calamine::Data::Int(42));
        assert_eq!(result, "42");
    }

    #[test]
    fn test_format_cell_value_float() {
        let result = format_cell_value(calamine::Data::Float(3.14159));
        assert!(result.starts_with("3.14"));
    }

    #[test]
    fn test_format_cell_value_bool() {
        let result = format_cell_value(calamine::Data::Bool(true));
        assert_eq!(result, "true");
    }
}

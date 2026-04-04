use crate::error::{Result, SidecarError};
use crate::document::validate_file_size;
use std::collections::HashMap;

#[derive(Debug, Clone)]
pub enum ChunkingStrategy {
    Fixed { chunk_size: usize },
    SlidingWindow { window_size: usize, step_size: usize },
    Paragraph,
    Sentence,
}

impl Default for ChunkingStrategy {
    fn default() -> Self {
        Self::Fixed { chunk_size: 512 }
    }
}

#[derive(Debug, Clone)]
pub struct TextChunk {
    pub text: String,
    pub index: usize,
    pub start_char: usize,
    pub end_char: usize,
    pub metadata: HashMap<String, String>,
}

pub struct TextChunker {
    strategy: ChunkingStrategy,
}

impl TextChunker {
    pub fn new(strategy: ChunkingStrategy) -> Self {
        Self { strategy }
    }

    pub fn chunk_text(&self, text: &str) -> Result<Vec<TextChunk>> {
        validate_file_size(text.len())?;

        match &self.strategy {
            ChunkingStrategy::Fixed { chunk_size } => {
                self.chunk_fixed(text, *chunk_size)
            }
            ChunkingStrategy::SlidingWindow { window_size, step_size } => {
                self.chunk_sliding_window(text, *window_size, *step_size)
            }
            ChunkingStrategy::Paragraph => {
                self.chunk_by_paragraph(text)
            }
            ChunkingStrategy::Sentence => {
                self.chunk_by_sentence(text)
            }
        }
    }

    fn chunk_fixed(&self, text: &str, chunk_size: usize) -> Result<Vec<TextChunk>> {
        let mut chunks = Vec::new();
        let mut index = 0;
        let mut start_char = 0;

        while start_char < text.len() {
            let end_char = (start_char + chunk_size).min(text.len());
            let chunk_text = text[start_char..end_char].to_string();

            chunks.push(TextChunk {
                text: chunk_text,
                index,
                start_char,
                end_char,
                metadata: HashMap::new(),
            });

            index += 1;
            start_char = end_char;
        }

        Ok(chunks)
    }

    fn chunk_sliding_window(&self, text: &str, window_size: usize, step_size: usize) -> Result<Vec<TextChunk>> {
        if step_size > window_size {
            return Err(SidecarError::InvalidRequest(
                "Step size cannot be larger than window size".to_string(),
            ));
        }

        let mut chunks = Vec::new();
        let mut index = 0;
        let mut start_char = 0;

        while start_char < text.len() {
            let end_char = (start_char + window_size).min(text.len());
            let chunk_text = text[start_char..end_char].to_string();

            chunks.push(TextChunk {
                text: chunk_text,
                index,
                start_char,
                end_char,
                metadata: HashMap::new(),
            });

            index += 1;
            start_char += step_size;
        }

        Ok(chunks)
    }

    fn chunk_by_paragraph(&self, text: &str) -> Result<Vec<TextChunk>> {
        let mut chunks = Vec::new();
        let mut index = 0;
        let mut start_char = 0;

        let paragraphs: Vec<&str> = text.split("\n\n").collect();

        for para in paragraphs {
            if !para.trim().is_empty() {
                let end_char = start_char + para.len();

                chunks.push(TextChunk {
                    text: para.to_string(),
                    index,
                    start_char,
                    end_char,
                    metadata: HashMap::new(),
                });

                index += 1;
                start_char = end_char + 2;
            }
        }

        Ok(chunks)
    }

    fn chunk_by_sentence(&self, text: &str) -> Result<Vec<TextChunk>> {
        let mut chunks = Vec::new();
        let mut index = 0;
        let mut current_start = 0;

        for (i, c) in text.char_indices() {
            if c == '.' || c == '!' || c == '?' {
                let end_char = i + 1;
                let chunk_text = text[current_start..end_char].to_string();

                if !chunk_text.trim().is_empty() {
                    chunks.push(TextChunk {
                        text: chunk_text,
                        index,
                        start_char: current_start,
                        end_char,
                        metadata: HashMap::new(),
                    });

                    index += 1;
                    current_start = end_char;
                }
            }
        }

        if current_start < text.len() {
            let chunk_text = text[current_start..].to_string();
            if !chunk_text.trim().is_empty() {
                chunks.push(TextChunk {
                    text: chunk_text,
                    index,
                    start_char: current_start,
                    end_char: text.len(),
                    metadata: HashMap::new(),
                });
            }
        }

        Ok(chunks)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_chunk_creation() {
        let chunker = TextChunker::new(ChunkingStrategy::Fixed { chunk_size: 100 });
        let result = chunker.chunk_text("Hello world");
        assert!(result.is_ok());
        let chunks = result.unwrap();
        assert!(chunks[0].start_char < chunks[0].end_char);
    }

    #[test]
    fn test_empty_text() {
        let chunker = TextChunker::new(ChunkingStrategy::default());
        let result = chunker.chunk_text("");
        assert!(result.is_ok());
        let chunks = result.unwrap();
        assert!(chunks.is_empty());
    }

    #[test]
    fn test_fixed_chunking() {
        let text = "This is a test text for chunking with fixed size chunks";
        let chunker = TextChunker::new(ChunkingStrategy::Fixed { chunk_size: 20 });
        let result = chunker.chunk_text(text);
        assert!(result.is_ok());
        let chunks = result.unwrap();
        assert!(chunks.len() > 1);
    }

    #[test]
    fn test_sliding_window() {
        let text = "This is a test for sliding window chunking with overlap between chunks";
        let chunker = TextChunker::new(ChunkingStrategy::SlidingWindow {
            window_size: 30,
            step_size: 10,
        });
        let result = chunker.chunk_text(text);
        assert!(result.is_ok());
        let chunks = result.unwrap();
        assert!(chunks.len() > 2);
        if chunks.len() > 1 {
            assert!(chunks[0].end_char > chunks[1].start_char);
        }
    }

    #[test]
    fn test_paragraph_chunking() {
        let text = "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.";
        let chunker = TextChunker::new(ChunkingStrategy::Paragraph);
        let result = chunker.chunk_text(text);
        assert!(result.is_ok());
        let chunks = result.unwrap();
        assert_eq!(chunks.len(), 3);
    }

    #[test]
    fn test_sentence_chunking() {
        let text = "First sentence. Second sentence! Third sentence? Fourth one.";
        let chunker = TextChunker::new(ChunkingStrategy::Sentence);
        let result = chunker.chunk_text(text);
        assert!(result.is_ok());
        let chunks = result.unwrap();
        assert_eq!(chunks.len(), 4);
    }

    #[test]
    fn test_large_text() {
        let text: String = (1..1000).map(|i| format!("Sentence {}. ", i)).collect();
        let chunker = TextChunker::new(ChunkingStrategy::Fixed { chunk_size: 100 });
        let result = chunker.chunk_text(&text);
        assert!(result.is_ok());
        let chunks = result.unwrap();
        assert!(chunks.len() > 10);
    }

    #[test]
    fn test_invalid_sliding_window() {
        let chunker = TextChunker::new(ChunkingStrategy::SlidingWindow {
            window_size: 10,
            step_size: 20,
        });
        let result = chunker.chunk_text("Test");
        assert!(result.is_err());
    }

    #[test]
    fn test_metadata_preservation() {
        let text = "Test chunk";
        let chunker = TextChunker::new(ChunkingStrategy::Fixed { chunk_size: 100 });
        let result = chunker.chunk_text(text);
        assert!(result.is_ok());
        let chunks = result.unwrap();
        assert!(chunks[0].metadata.is_empty());
    }

    #[test]
    fn test_unicode_text() {
        let text = "Hello 世界. This is a test. Unicode: 日本語";
        let chunker = TextChunker::new(ChunkingStrategy::Sentence);
        let result = chunker.chunk_text(text);
        assert!(result.is_ok());
        let chunks = result.unwrap();
        assert!(chunks.len() >= 2);
    }

    #[test]
    fn test_chunk_indices() {
        let text = "First. Second. Third.";
        let chunker = TextChunker::new(ChunkingStrategy::Sentence);
        let result = chunker.chunk_text(text);
        assert!(result.is_ok());
        let chunks = result.unwrap();
        assert!(chunks[0].start_char < chunks[0].end_char);
    }
}

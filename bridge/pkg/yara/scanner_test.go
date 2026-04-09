package yara

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func testRuleFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join("testdata", "test_rules.yar")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("test rules file not found: %s", p)
	}
	return p
}

func TestScanFile_CleanPDF(t *testing.T) {
	ruleFile := testRuleFile(t)
	if err := InitYARA(ruleFile); err != nil {
		t.Fatalf("InitYARA failed: %v", err)
	}

	clean, err := ScanFileForMalware(filepath.Join("testdata", "clean.txt"))
	if err != nil {
		t.Fatalf("ScanFileForMalware returned error for clean file: %v", err)
	}
	if !clean {
		t.Error("expected clean=true for clean text file, got false")
	}
}

func TestScanFile_EICAR(t *testing.T) {
	ruleFile := testRuleFile(t)
	if err := InitYARA(ruleFile); err != nil {
		t.Fatalf("InitYARA failed: %v", err)
	}

	clean, err := ScanFileForMalware(filepath.Join("testdata", "eicar.txt"))
	if err != nil {
		t.Fatalf("ScanFileForMalware returned error: %v", err)
	}
	if clean {
		t.Error("expected clean=false for EICAR test file, got true")
	}
}

func TestScanFile_MacroDOCX(t *testing.T) {
	ruleFile := testRuleFile(t)
	if err := InitYARA(ruleFile); err != nil {
		t.Fatalf("InitYARA failed: %v", err)
	}

	tmpDir := t.TempDir()
	macroFile := filepath.Join(tmpDir, "macro_test.txt")
	if err := os.WriteFile(macroFile, []byte("Sub AutoOpen()\nMsgBox \"test\"\nEnd Sub"), 0644); err != nil {
		t.Fatalf("failed to create macro test file: %v", err)
	}

	clean, err := ScanFileForMalware(macroFile)
	if err != nil {
		t.Fatalf("ScanFileForMalware returned error: %v", err)
	}
	if clean {
		t.Error("expected clean=false for file with macro pattern, got true")
	}
}

func TestScanFile_NotFound(t *testing.T) {
	ruleFile := testRuleFile(t)
	if err := InitYARA(ruleFile); err != nil {
		t.Fatalf("InitYARA failed: %v", err)
	}

	_, err := ScanFileForMalware("/nonexistent/path/to/file.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestScanFile_EmptyFile(t *testing.T) {
	ruleFile := testRuleFile(t)
	if err := InitYARA(ruleFile); err != nil {
		t.Fatalf("InitYARA failed: %v", err)
	}

	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	clean, err := ScanFileForMalware(emptyFile)
	if err != nil {
		t.Fatalf("ScanFileForMalware returned error for empty file: %v", err)
	}
	if !clean {
		t.Error("expected clean=true for empty file, got false")
	}
}

func TestInitYARA_ValidRules(t *testing.T) {
	ruleFile := testRuleFile(t)
	err := InitYARA(ruleFile)
	if err != nil {
		t.Fatalf("InitYARA with valid rules should succeed, got: %v", err)
	}
	if compiledRules == nil {
		t.Error("expected compiledRules to be non-nil after InitYARA")
	}
}

func TestInitYARA_InvalidRules(t *testing.T) {
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.yar")
	if err := os.WriteFile(invalidFile, []byte("rule broken { condition: ???"), 0644); err != nil {
		t.Fatalf("failed to create invalid rules file: %v", err)
	}

	err := InitYARA(invalidFile)
	if err == nil {
		t.Error("expected error for invalid YARA rules, got nil")
	}
}

func TestInitYARA_MissingFile(t *testing.T) {
	err := InitYARA("/nonexistent/rules.yar")
	if err == nil {
		t.Error("expected error for missing rule file, got nil")
	}
}

func TestScanFile_ConcurrentSafety(t *testing.T) {
	ruleFile := testRuleFile(t)
	if err := InitYARA(ruleFile); err != nil {
		t.Fatalf("InitYARA failed: %v", err)
	}

	var wg sync.WaitGroup
	const goroutines = 20

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			target := "testdata/clean.txt"
			if i%2 == 0 {
				target = "testdata/eicar.txt"
			}
			clean, err := ScanFileForMalware(target)
			if err != nil {
				t.Errorf("concurrent scan error for %s: %v", target, err)
				return
			}
			if i%2 == 0 && clean {
				t.Error("concurrent scan: expected infected for EICAR")
			}
			if i%2 != 0 && !clean {
				t.Error("concurrent scan: expected clean for clean.txt")
			}
		}()
	}

	wg.Wait()
}

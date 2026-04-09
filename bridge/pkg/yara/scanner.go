package yara

import (
	"fmt"
	"log"
	"os"
	"sync"

	yara "github.com/hillu/go-yara/v4"
)

var (
	compiledRules *yara.Rules
	rulesMu       sync.RWMutex
)

func InitYARA(ruleFile string) error {
	f, err := os.Open(ruleFile)
	if err != nil {
		return fmt.Errorf("failed to open YARA rules file: %w", err)
	}
	defer f.Close()

	compiler, err := yara.NewCompiler()
	if err != nil {
		return fmt.Errorf("failed to create YARA compiler: %w", err)
	}

	if err := compiler.AddFile(f, ""); err != nil {
		return fmt.Errorf("failed to add YARA rules: %w", err)
	}

	rules, err := compiler.GetRules()
	if err != nil {
		return fmt.Errorf("failed to compile YARA rules: %w", err)
	}

	rulesMu.Lock()
	compiledRules = rules
	rulesMu.Unlock()

	return nil
}

func ScanFileForMalware(filePath string) (bool, error) {
	rulesMu.RLock()
	rules := compiledRules
	rulesMu.RUnlock()

	if rules == nil {
		return false, fmt.Errorf("YARA not initialized")
	}

	var matchRules yara.MatchRules

	err := rules.ScanFile(filePath, 0, 0, &matchRules)
	if err != nil {
		return false, err
	}

	if len(matchRules) > 0 {
		for _, m := range matchRules {
			log.Printf("SECURITY: YARA rule matched: %s in %s", m.Rule, filePath)
		}
		return false, nil
	}

	return true, nil
}

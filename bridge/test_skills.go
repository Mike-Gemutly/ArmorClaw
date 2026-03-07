package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/armorclaw/bridge/internal/skills"
)

func main() {
	fmt.Println("=== Skills Integration Test ===")
	
	// Test 1: Skill Registry
	fmt.Println("1. Testing Skill Registry...")
	registry := skills.NewRegistry()
	
	// Load skills
	if err := registry.ScanSkills("../container/openclaw-src/skills"); err != nil {
		log.Fatalf("Failed to load skills: %v", err)
	}
	
	// List available skills
	skillList := registry.GetEnabled()
	fmt.Printf("   Loaded %d skills:\n", len(skillList))
	for _, skill := range skillList {
		fmt.Printf("   - %s (domain: %s, risk: %s)\n", skill.Name, skill.Domain, skill.Risk)
	}
	
	// Test 2: Get specific skill (weather)
	fmt.Println("\n2. Testing weather skill...")
	weatherSkill, exists := registry.GetSkill("weather")
	if !exists {
		log.Fatalf("weather skill not found")
	}
	
	fmt.Printf("   Skill: %s\n", weatherSkill.Name)
	fmt.Printf("   Description: %s\n", weatherSkill.Description)
	fmt.Printf("   Domain: %s\n", weatherSkill.Domain)
	fmt.Printf("   Risk: %s\n", weatherSkill.Risk)
	fmt.Printf("   Parameters: %v\n", weatherSkill.Parameters)
	
	// Test 3: Generate OpenAI schema
	fmt.Println("\n3. Testing OpenAI schema generation...")
	schemaGen := skills.NewSchemaGenerator(registry)
	schema, err := schemaGen.GenerateSchema(weatherSkill)
	if err != nil {
		log.Fatalf("Failed to generate schema: %v", err)
	}
	
	fmt.Printf("   Schema type: %s\n", schema.Type)
	fmt.Printf("   Function name: %s\n", schema.Function.Name)
	fmt.Printf("   Function description: %s\n", schema.Function.Description)
	
	// Test 4: Policy Validation
	fmt.Println("\n4. Testing policy validation...")
	policy, exists := skills.GetPolicy("weather")
	if !exists {
		log.Fatalf("weather policy not found")
	}
	
	fmt.Printf("   weather policy - Risk: %s, AutoExecute: %v, Timeout: %v\n", 
		policy.Risk, policy.AutoExecute, policy.Timeout)
	
	// Test 5: Router
	fmt.Println("\n5. Testing skill router...")
	router := skills.NewRouter()
	
	route := router.Route("weather")
	if route == "" {
		log.Fatalf("Failed to route weather skill")
	}
	fmt.Printf("   ✓ weather routed to domain: %s\n", route)
	
	// Test 6: Executor (dry run)
	fmt.Println("\n6. Testing executor (dry run)...")
	executor := skills.NewSkillExecutor()
	
	// Load skills into executor
	if err := executor.LoadSkills("../container/openclaw-src/skills"); err != nil {
		log.Fatalf("Failed to load skills into executor: %v", err)
	}
	
	ctx := context.Background()
	params := map[string]interface{}{
		"location": "New York, NY",
	}
	
	// Execute weather skill (dry run)
	result, err := executor.ExecuteSkill(ctx, "weather", params)
	if err != nil {
		log.Fatalf("Failed to execute weather: %v", err)
	}
	
	fmt.Printf("   Skill execution result type: %T\n", result)
	if resultBytes, err := json.MarshalIndent(result, "   ", "  "); err == nil {
		fmt.Printf("   Result: %s\n", string(resultBytes))
	}
	
	fmt.Println("\n=== All Tests Passed! ===")
	fmt.Println("Skills engine is fully functional.")
}
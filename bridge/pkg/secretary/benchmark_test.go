package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

//=============================================================================
// Benchmark: Cold Dispatch
//=============================================================================

// BenchmarkColdDispatch measures the full cold dispatch path:
// tick → ListDueTasks → coldDispatch → factory.Spawn → updateAfterDispatch
func BenchmarkColdDispatch(b *testing.B) {
	store := newSchedulerTestStore()
	now := time.Now()
	store.scheduledTasks["bench-task"] = &ScheduledTask{
		ID:             "bench-task",
		DefinitionID:   "def-bench",
		CronExpression: "0 * * * *",
		IsActive:       true,
		NextRun:        &now,
		CreatedBy:      "@bench:example.com",
	}

	factory := &mockSchedulerFactory{
		spawnResult: &SpawnResultRef{
			InstanceID: "inst-bench",
			RoomID:     "!bench-room:example.com",
		},
	}
	matrix := &mockSchedulerMatrix{}

	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Reset next_run so the task is always due
		store.Lock()
		n := time.Now()
		store.scheduledTasks["bench-task"].NextRun = &n
		store.scheduledTasks["bench-task"].IsActive = true
		store.Unlock()

		scheduler.tick()
	}
}

// BenchmarkColdDispatch_ParallelTasks measures cold dispatch with multiple due tasks.
func BenchmarkColdDispatch_ParallelTasks(b *testing.B) {
	const taskCount = 10

	store := newSchedulerTestStore()
	now := time.Now()
	for i := 0; i < taskCount; i++ {
		id := fmt.Sprintf("bench-task-%d", i)
		store.scheduledTasks[id] = &ScheduledTask{
			ID:             id,
			DefinitionID:   fmt.Sprintf("def-bench-%d", i),
			CronExpression: "0 * * * *",
			IsActive:       true,
			NextRun:        &now,
			CreatedBy:      "@bench:example.com",
		}
	}

	factory := &mockSchedulerFactory{
		spawnResult: &SpawnResultRef{
			InstanceID: "inst-bench",
			RoomID:     "!bench-room:example.com",
		},
	}
	matrix := &mockSchedulerMatrix{}
	scheduler := NewTaskScheduler(store, factory, matrix, nil, nil, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		store.Lock()
		n := time.Now()
		for j := 0; j < taskCount; j++ {
			id := fmt.Sprintf("bench-task-%d", j)
			store.scheduledTasks[id].NextRun = &n
			store.scheduledTasks[id].IsActive = true
		}
		store.Unlock()

		scheduler.tick()
	}
}

//=============================================================================
// Benchmark: Inter-Step Data Propagation
//=============================================================================

// BenchmarkInterStepDataPropagation measures the overhead of resolving
// {{steps.X.data.Y}} template references and injecting into step config.
func BenchmarkInterStepDataPropagation(b *testing.B) {
	accumulatedData := map[string]map[string]any{
		"step_1": {
			"order_id":  "ORD-12345",
			"total":     "99.99",
			"currency":  "USD",
			"customer":  "alice@example.com",
			"item_count": "3",
		},
		"step_2": {
			"tracking_number": "TRACK-67890",
			"carrier":         "FedEx",
			"estimated_days":  "3",
		},
	}

	input := map[string]any{
		"order_ref":   "{{steps.step_1.data.order_id}}",
		"total_ref":   "{{steps.step_1.data.total}}",
		"track_ref":   "{{steps.step_2.data.tracking_number}}",
		"static_key":  "unchanged_value",
		"carrier_ref": "{{steps.step_2.data.carrier}}",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resolveStepInput(input, accumulatedData)
	}
}

// BenchmarkInterStepDataPropagation_LargeMap measures data propagation
// overhead with many accumulated steps and input references.
func BenchmarkInterStepDataPropagation_LargeMap(b *testing.B) {
	// Simulate 20 steps each with 10 data keys
	accumulatedData := make(map[string]map[string]any, 20)
	for i := 0; i < 20; i++ {
		stepData := make(map[string]any, 10)
		for j := 0; j < 10; j++ {
			stepData[fmt.Sprintf("key_%d", j)] = fmt.Sprintf("value_%d_%d", i, j)
		}
		accumulatedData[fmt.Sprintf("step_%d", i)] = stepData
	}

	// Input referencing 15 of those keys
	input := make(map[string]any, 15)
	for i := 0; i < 15; i++ {
		key := fmt.Sprintf("ref_%d", i)
		input[key] = fmt.Sprintf("{{steps.step_%d.data.key_%d}}", i%20, i%10)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resolveStepInput(input, accumulatedData)
	}
}

// BenchmarkInjectPrevStepData measures JSON injection of resolved step data
// into step config.
func BenchmarkInjectPrevStepData(b *testing.B) {
	config := json.RawMessage(`{"action":"fill_form","url":"https://example.com"}`)
	input := map[string]any{
		"order_id": "ORD-12345",
		"total":    "99.99",
		"currency": "USD",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		injectPrevStepData(config, input)
	}
}

// BenchmarkResolveTemplateString measures the regex-based template resolution.
func BenchmarkResolveTemplateString(b *testing.B) {
	accumulatedData := map[string]map[string]any{
		"step_1": {"key_a": "value_a", "key_b": "value_b"},
		"step_2": {"key_c": "value_c"},
	}

	template := "order={{steps.step_1.data.key_a}}&total={{steps.step_1.data.key_b}}&track={{steps.step_2.data.key_c}}"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resolveTemplateString(template, accumulatedData)
	}
}

//=============================================================================
// Benchmark: Sequential Workflow Execution
//=============================================================================

// benchStoreWithTemplate creates a store pre-populated with a workflow and
// template suitable for benchmarking.
func benchStoreWithTemplate(stepCount int) *orchestratorTestStore {
	store := newOrchestratorTestStore()

	steps := make([]WorkflowStep, stepCount)
	for i := 0; i < stepCount; i++ {
		steps[i] = WorkflowStep{
			StepID:   fmt.Sprintf("step_%d", i),
			Order:    i,
			Type:     StepAction,
			Name:     fmt.Sprintf("Step %d", i),
			AgentIDs: []string{fmt.Sprintf("agent-%d", i)},
		}
	}

	store.templates["bench-tmpl"] = &TaskTemplate{
		ID:        "bench-tmpl",
		Name:      "Bench Template",
		IsActive:  true,
		CreatedBy: "@bench:example.com",
		Steps:     steps,
	}

	return store
}

// BenchmarkSequentialWorkflow measures the overhead of the orchestrator's
// sequential workflow lifecycle: StartWorkflow → executeWorkflow.
// The workflow ticker runs once per second; we measure the full
// StartWorkflow path (store reads, state transitions, event emission).
func BenchmarkSequentialWorkflow(b *testing.B) {
	store := benchStoreWithTemplate(5)
	emitter := newMockEventEmitter()

	orch, err := NewWorkflowOrchestrator(OrchestratorConfig{
		Store:    store,
		EventBus: emitter,
	})
	if err != nil {
		b.Fatal(err)
	}

	// Pre-create workflows
	workflowBase := &Workflow{
		TemplateID: "bench-tmpl",
		Name:       "Bench Workflow",
		Status:     StatusPending,
		CreatedBy:  "@bench:example.com",
		StartedAt:  time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wfID := fmt.Sprintf("wf-seq-%d", i)
		wf := *workflowBase
		wf.ID = wfID
		if err := store.CreateWorkflow(context.Background(), &wf); err != nil {
			b.Fatal(err)
		}
		if err := orch.StartWorkflow(wfID); err != nil {
			b.Fatal(err)
		}
		// Clean up for next iteration
		orch.CancelWorkflow(wfID, "benchmark cleanup")
	}
}

// BenchmarkSequentialWorkflowLifecycle measures the full lifecycle:
// create → start → advance through steps → complete.
func BenchmarkSequentialWorkflowLifecycle(b *testing.B) {
	store := benchStoreWithTemplate(3)
	emitter := newMockEventEmitter()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		orch, _ := NewWorkflowOrchestrator(OrchestratorConfig{
			Store:    store,
			EventBus: emitter,
		})

		wfID := fmt.Sprintf("wf-life-%d", i)
		wf := &Workflow{
			ID:         wfID,
			TemplateID: "bench-tmpl",
			Name:       "Bench Lifecycle",
			Status:     StatusPending,
			CreatedBy:  "@bench:example.com",
			StartedAt:  time.Now(),
		}
		store.CreateWorkflow(context.Background(), wf)
		orch.StartWorkflow(wfID)

		// Advance through each step
		tmpl := store.templates["bench-tmpl"]
		for _, step := range tmpl.Steps {
			orch.AdvanceWorkflow(wfID, step.StepID)
		}

		// Complete should happen on last advance
		// Cancel to clean up active workflow
		orch.CancelWorkflow(wfID, "bench")
	}
}

//=============================================================================
// Benchmark: Parallel Workflow Identification
//=============================================================================

// BenchmarkIdentifyParallelGroups measures the overhead of scanning steps
// to find split→merge parallel groups.
func BenchmarkIdentifyParallelGroups(b *testing.B) {
	steps := make([]WorkflowStep, 20)
	// 0: action
	steps[0] = WorkflowStep{StepID: "s0", Order: 0, Type: StepAction, Name: "init"}
	// 1: parallel_split
	steps[1] = WorkflowStep{StepID: "s1", Order: 1, Type: StepParallelSplit, Name: "split"}
	// 2-5: branches
	for i := 2; i <= 5; i++ {
		steps[i] = WorkflowStep{
			StepID:   fmt.Sprintf("branch_%d", i),
			Order:    i,
			Type:     StepAction,
			Name:     fmt.Sprintf("Branch %d", i),
			AgentIDs: []string{fmt.Sprintf("agent-%d", i)},
		}
	}
	// 6: parallel_merge
	steps[6] = WorkflowStep{StepID: "s6", Order: 6, Type: StepParallelMerge, Name: "merge"}
	// 7-19: more sequential steps
	for i := 7; i < 20; i++ {
		steps[i] = WorkflowStep{
			StepID:   fmt.Sprintf("s%d", i),
			Order:    i,
			Type:     StepAction,
			Name:     fmt.Sprintf("Step %d", i),
			AgentIDs: []string{fmt.Sprintf("agent-%d", i)},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		IdentifyParallelGroups(steps)
	}
}

// BenchmarkIdentifyParallelGroups_NoParallel measures overhead when no
// parallel steps exist (pure sequential).
func BenchmarkIdentifyParallelGroups_NoParallel(b *testing.B) {
	steps := make([]WorkflowStep, 20)
	for i := 0; i < 20; i++ {
		steps[i] = WorkflowStep{
			StepID:   fmt.Sprintf("s%d", i),
			Order:    i,
			Type:     StepAction,
			Name:     fmt.Sprintf("Step %d", i),
			AgentIDs: []string{fmt.Sprintf("agent-%d", i)},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		IdentifyParallelGroups(steps)
	}
}

//=============================================================================
// Benchmark: Dependency Validation
//=============================================================================

// BenchmarkDependencyValidation measures template validation with
// topological sort.
func BenchmarkDependencyValidation(b *testing.B) {
	validator := NewDependencyValidator()

	steps := make([]WorkflowStep, 50)
	for i := 0; i < 50; i++ {
		nextID := ""
		if i < 49 {
			nextID = fmt.Sprintf("s%d", i+1)
		}
		steps[i] = WorkflowStep{
			StepID:     fmt.Sprintf("s%d", i),
			Order:      i,
			Type:       StepAction,
			Name:       fmt.Sprintf("Step %d", i),
			NextStepID: nextID,
			AgentIDs:   []string{fmt.Sprintf("agent-%d", i)},
		}
	}

	template := &TaskTemplate{
		ID:        "bench-validate",
		Steps:     steps,
		IsActive:  true,
		CreatedBy: "@bench:example.com",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		validator.ValidateTemplate(template)
	}
}

//=============================================================================
// Benchmark: Event Emission
//=============================================================================

// BenchmarkEventEmission measures the mockEventEmitter's throughput to
// establish a baseline for event emission overhead.
func BenchmarkEventEmission(b *testing.B) {
	emitter := newMockEventEmitter()
	wf := &Workflow{
		ID:         "wf-bench",
		TemplateID: "tmpl-bench",
		Status:     StatusRunning,
		CreatedBy:  "@bench:example.com",
		StartedAt:  time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		emitter.EmitStarted(wf)
		emitter.EmitProgress(wf, "step_1", "Step 1", 0.5)
		emitter.EmitCompleted(wf, "done")
	}
}

// BenchmarkEventEmission_Failed measures emission for the failed workflow path.
func BenchmarkEventEmission_Failed(b *testing.B) {
	emitter := newMockEventEmitter()
	wf := &Workflow{
		ID:         "wf-bench-fail",
		TemplateID: "tmpl-bench",
		Status:     StatusFailed,
		CreatedBy:  "@bench:example.com",
		StartedAt:  time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		emitter.EmitFailed(wf, "step_3", fmt.Errorf("benchmark error"), false)
	}
}

package team

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

func openTestDB(t *testing.T) *TeamStore {
	t.Helper()
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&_foreign_keys=ON")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := NewTeamStoreFromDB(db)
	if err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return store
}

func validTeam(id, name string) *Team {
	return &Team{
		ID:             id,
		Name:           name,
		LifecycleState: LifecycleActive,
	}
}

func validMember(teamID, agentID, role string) *TeamMember {
	return &TeamMember{
		TeamID:   teamID,
		AgentID:  agentID,
		RoleName: role,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateTeam_GetTeam(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	team := validTeam("t1", "Alpha Team")
	team.SharedContext = "research context"
	team.Budgets = &TeamBudgets{MaxTokenUsage: 1000, MaxCost: 5.0}

	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	got, err := store.GetTeam(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}

	if got.ID != "t1" {
		t.Errorf("ID = %q, want %q", got.ID, "t1")
	}
	if got.Name != "Alpha Team" {
		t.Errorf("Name = %q, want %q", got.Name, "Alpha Team")
	}
	if got.SharedContext != "research context" {
		t.Errorf("SharedContext = %q, want %q", got.SharedContext, "research context")
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}
	if got.LifecycleState != LifecycleActive {
		t.Errorf("LifecycleState = %q, want %q", got.LifecycleState, LifecycleActive)
	}
	if got.Budgets == nil {
		t.Fatal("Budgets is nil")
	}
	if got.Budgets.MaxTokenUsage != 1000 {
		t.Errorf("Budgets.MaxTokenUsage = %d, want 1000", got.Budgets.MaxTokenUsage)
	}
}

func TestGetTeam_NotFound(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	_, err := store.GetTeam(ctx, "nonexistent")
	if err != ErrTeamNotFound {
		t.Fatalf("error = %v, want ErrTeamNotFound", err)
	}
}

func TestCreateTeam_Validation(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	bad := &Team{Name: "x", LifecycleState: LifecycleActive}
	if err := store.CreateTeam(ctx, bad); err == nil {
		t.Fatal("expected validation error for empty ID")
	}
}

func TestListTeams(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		tm := validTeam(fmt.Sprintf("t%d", i), fmt.Sprintf("Team %d", i))
		if err := store.CreateTeam(ctx, tm); err != nil {
			t.Fatalf("CreateTeam %d: %v", i, err)
		}
	}

	teams, err := store.ListTeams(ctx)
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 3 {
		t.Fatalf("len(teams) = %d, want 3", len(teams))
	}
}

func TestListTeams_Empty(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	teams, err := store.ListTeams(ctx)
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 0 {
		t.Fatalf("len(teams) = %d, want 0", len(teams))
	}
}

func TestUpdateTeam_OptimisticLock(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	team := validTeam("t1", "V1")
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	got, err := store.GetTeam(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	got.Name = "V2"
	if err := store.UpdateTeam(ctx, got); err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}
	if got.Version != 2 {
		t.Errorf("Version after update = %d, want 2", got.Version)
	}

	stale := validTeam("t1", "Stale")
	stale.Version = 1
	if err := store.UpdateTeam(ctx, stale); err != ErrVersionConflict {
		t.Fatalf("stale update error = %v, want ErrVersionConflict", err)
	}

	latest, _ := store.GetTeam(ctx, "t1")
	if latest.Name != "V2" {
		t.Errorf("Name = %q, want %q after conflict", latest.Name, "V2")
	}
}

func TestAddMember_RemoveMember(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	team := validTeam("t1", "Research")
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	m1 := validMember("t1", "agent-a", "browser_specialist")
	m1.AllowedTools = []string{"browse", "screenshot"}
	m1.AllowedSecretPrefixes = []string{"web/"}
	m1.Priority = 3

	if err := store.AddMember(ctx, m1); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	m2 := validMember("t1", "agent-b", "doc_analyst")
	if err := store.AddMember(ctx, m2); err != nil {
		t.Fatalf("AddMember second: %v", err)
	}

	got, err := store.GetTeam(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if len(got.Members) != 2 {
		t.Fatalf("members count = %d, want 2", len(got.Members))
	}

	var foundA bool
	for _, m := range got.Members {
		if m.AgentID == "agent-a" {
			foundA = true
			if len(m.AllowedTools) != 2 {
				t.Errorf("AllowedTools len = %d, want 2", len(m.AllowedTools))
			}
			if m.Priority != 3 {
				t.Errorf("Priority = %d, want 3", m.Priority)
			}
		}
	}
	if !foundA {
		t.Error("agent-a not found in members")
	}

	if err := store.RemoveMember(ctx, "t1", "agent-a"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	got, _ = store.GetTeam(ctx, "t1")
	if len(got.Members) != 1 {
		t.Fatalf("members after remove = %d, want 1", len(got.Members))
	}
	if got.Members[0].AgentID != "agent-b" {
		t.Errorf("remaining member = %q, want agent-b", got.Members[0].AgentID)
	}
}

func TestRemoveMember_NotFound(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	if err := store.RemoveMember(ctx, "t1", "ghost"); err != ErrMemberNotFound {
		t.Fatalf("error = %v, want ErrMemberNotFound", err)
	}
}

func TestRemoveLastMember_AutoDissolve(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	team := validTeam("t1", "Solo")
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	m := validMember("t1", "only-agent", "team_lead")
	if err := store.AddMember(ctx, m); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	if err := store.RemoveMember(ctx, "t1", "only-agent"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	got, err := store.GetTeam(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got.LifecycleState != LifecycleDissolved {
		t.Errorf("LifecycleState = %q, want dissolved", got.LifecycleState)
	}
	if len(got.Members) != 0 {
		t.Errorf("Members len = %d, want 0", len(got.Members))
	}
}

func TestDissolveTeam(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	team := validTeam("t1", "Doomed")
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	m := validMember("t1", "agent-x", "browser_specialist")
	if err := store.AddMember(ctx, m); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	if err := store.DissolveTeam(ctx, "t1"); err != nil {
		t.Fatalf("DissolveTeam: %v", err)
	}

	got, _ := store.GetTeam(ctx, "t1")
	if got.LifecycleState != LifecycleDissolved {
		t.Errorf("LifecycleState = %q, want dissolved", got.LifecycleState)
	}

	// Data preserved for audit.
	if len(got.Members) != 1 {
		t.Errorf("Members len = %d, want 1 (preserved for audit)", len(got.Members))
	}
}

func TestDissolveTeam_Idempotent(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	team := validTeam("t1", "Double")
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	if err := store.DissolveTeam(ctx, "t1"); err != nil {
		t.Fatalf("first dissolve: %v", err)
	}
	if err := store.DissolveTeam(ctx, "t1"); err != nil {
		t.Fatalf("second dissolve (idempotent): %v", err)
	}
}

func TestDissolveTeam_NotFound(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	err := store.DissolveTeam(ctx, "ghost")
	if err != ErrTeamNotFound {
		t.Fatalf("error = %v, want ErrTeamNotFound", err)
	}
}

func TestAddMember_DissolvedTeam(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	team := validTeam("t1", "Dead")
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if err := store.DissolveTeam(ctx, "t1"); err != nil {
		t.Fatalf("DissolveTeam: %v", err)
	}

	m := validMember("t1", "late-agent", "browser_specialist")
	err := store.AddMember(ctx, m)
	if err == nil {
		t.Fatal("expected error adding member to dissolved team")
	}
}

func TestAddMember_InvalidTeamRef(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	m := validMember("nonexistent", "agent-1", "browser_specialist")
	err := store.AddMember(ctx, m)
	if err != ErrTeamNotFound {
		t.Fatalf("error = %v, want ErrTeamNotFound", err)
	}
}

func TestAgentInMultipleTeams(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	t1 := validTeam("team-1", "Alpha")
	t2 := validTeam("team-2", "Beta")
	if err := store.CreateTeam(ctx, t1); err != nil {
		t.Fatalf("CreateTeam t1: %v", err)
	}
	if err := store.CreateTeam(ctx, t2); err != nil {
		t.Fatalf("CreateTeam t2: %v", err)
	}

	m1 := validMember("team-1", "shared-agent", "browser_specialist")
	m2 := validMember("team-2", "shared-agent", "team_lead")

	if err := store.AddMember(ctx, m1); err != nil {
		t.Fatalf("AddMember t1: %v", err)
	}
	if err := store.AddMember(ctx, m2); err != nil {
		t.Fatalf("AddMember t2: %v", err)
	}

	got1, _ := store.GetTeam(ctx, "team-1")
	got2, _ := store.GetTeam(ctx, "team-2")

	if len(got1.Members) != 1 || got1.Members[0].AgentID != "shared-agent" {
		t.Error("shared-agent not in team-1")
	}
	if len(got2.Members) != 1 || got2.Members[0].AgentID != "shared-agent" {
		t.Error("shared-agent not in team-2")
	}

	// Removing from team-1 should not affect team-2.
	if err := store.RemoveMember(ctx, "team-1", "shared-agent"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	got1, _ = store.GetTeam(ctx, "team-1")
	if got1.LifecycleState != LifecycleDissolved {
		t.Error("team-1 should be auto-dissolved after removing last member")
	}

	got2, _ = store.GetTeam(ctx, "team-2")
	if len(got2.Members) != 1 {
		t.Error("team-2 should still have the shared agent")
	}
}

func TestNewTeamStore_FileBased(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "teams.db")

	store, err := NewTeamStore(dbPath, "")
	if err != nil {
		t.Fatalf("NewTeamStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	team := validTeam("file-test", "FileTeam")
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	got, err := store.GetTeam(ctx, "file-test")
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got.Name != "FileTeam" {
		t.Errorf("Name = %q, want FileTeam", got.Name)
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("db file not created: %v", err)
	}
}

func TestConcurrentCreateTeams(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "concurrent.db")

	store, err := NewTeamStore(dbPath, "")
	if err != nil {
		t.Fatalf("NewTeamStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tm := validTeam(fmt.Sprintf("concurrent-%d", idx), fmt.Sprintf("Team %d", idx))
			if err := store.CreateTeam(ctx, tm); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent CreateTeam: %v", err)
	}

	teams, _ := store.ListTeams(ctx)
	if len(teams) != 10 {
		t.Errorf("len(teams) = %d, want 10", len(teams))
	}
}

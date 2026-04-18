package team

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

func newTestService(t *testing.T) (*Service, *TeamStore) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := NewTeamStoreFromDB(db)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}

	return NewTeamService(store), store
}

func TestService_CreateTeam(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "alpha", "")
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if team.ID == "" {
		t.Fatal("expected non-empty team ID")
	}
	if team.Name != "alpha" {
		t.Fatalf("expected name=alpha, got %q", team.Name)
	}
	if team.LifecycleState != LifecycleActive {
		t.Fatalf("expected active lifecycle, got %q", team.LifecycleState)
	}

	got, err := svc.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatalf("GetTeam after create: %v", err)
	}
	if got.Name != "alpha" {
		t.Fatalf("retrieved name mismatch: %q", got.Name)
	}
}

func TestService_GetTeam_NotFound(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.GetTeam(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent team")
	}
}

func TestService_ListTeams(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateTeam(ctx, "a", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateTeam(ctx, "b", ""); err != nil {
		t.Fatal(err)
	}

	teams, err := svc.ListTeams(ctx)
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(teams))
	}
}

func TestService_AddMember(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "team1", "")
	if err != nil {
		t.Fatal(err)
	}

	m, err := svc.AddMember(ctx, team.ID, "agent-1", "browser_specialist")
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if m.AgentID != "agent-1" {
		t.Fatalf("expected agent_id=agent-1, got %q", m.AgentID)
	}
	if m.RoleName != "browser_specialist" {
		t.Fatalf("expected role=browser_specialist, got %q", m.RoleName)
	}

	got, err := svc.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(got.Members))
	}
}

func TestService_AddMember_InvalidRole(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "team1", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.AddMember(ctx, team.ID, "agent-1", "nonexistent_role")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestService_AddMember_DuplicateTeamLead(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "team1", "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.AddMember(ctx, team.ID, "lead-1", "team_lead"); err != nil {
		t.Fatal(err)
	}

	_, err = svc.AddMember(ctx, team.ID, "lead-2", "team_lead")
	if err == nil {
		t.Fatal("expected error for duplicate team_lead")
	}
}

func TestService_RemoveMember(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "team1", "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.AddMember(ctx, team.ID, "agent-1", "browser_specialist"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddMember(ctx, team.ID, "agent-2", "doc_analyst"); err != nil {
		t.Fatal(err)
	}

	if err := svc.RemoveMember(ctx, team.ID, "agent-1"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	got, err := svc.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Members) != 1 {
		t.Fatalf("expected 1 member after removal, got %d", len(got.Members))
	}
}

func TestService_DissolveTeam(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "team1", "")
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.DissolveTeam(ctx, team.ID); err != nil {
		t.Fatalf("DissolveTeam: %v", err)
	}

	got, err := svc.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LifecycleState != LifecycleDissolved {
		t.Fatalf("expected dissolved, got %q", got.LifecycleState)
	}

	_, err = svc.AddMember(ctx, team.ID, "agent-1", "browser_specialist")
	if err == nil {
		t.Fatal("expected error adding member to dissolved team")
	}
}

func TestService_AssignRole(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "team1", "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.AddMember(ctx, team.ID, "agent-1", "browser_specialist"); err != nil {
		t.Fatal(err)
	}

	if err := svc.AssignRole(ctx, team.ID, "agent-1", "doc_analyst"); err != nil {
		t.Fatalf("AssignRole: %v", err)
	}

	got, err := svc.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(got.Members))
	}
	if got.Members[0].RoleName != "doc_analyst" {
		t.Fatalf("expected role=doc_analyst, got %q", got.Members[0].RoleName)
	}
}

func TestService_AssignRole_InvalidRole(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "team1", "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.AddMember(ctx, team.ID, "agent-1", "browser_specialist"); err != nil {
		t.Fatal(err)
	}

	err = svc.AssignRole(ctx, team.ID, "agent-1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestService_GetCapabilitiesForMember(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "team1", "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.AddMember(ctx, team.ID, "agent-1", "browser_specialist"); err != nil {
		t.Fatal(err)
	}

	caps, err := svc.GetCapabilitiesForMember(ctx, team.ID, "agent-1")
	if err != nil {
		t.Fatalf("GetCapabilitiesForMember: %v", err)
	}
	if !caps["browser.browse"] {
		t.Fatal("expected browser.browse capability")
	}
	if !caps["browser.extract"] {
		t.Fatal("expected browser.extract capability")
	}
	if caps["doc.ingest"] {
		t.Fatal("did not expect doc.ingest capability for browser_specialist")
	}
}

func TestService_GetCapabilitiesForMember_NotMember(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "team1", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.GetCapabilitiesForMember(ctx, team.ID, "nonexistent-agent")
	if err == nil {
		t.Fatal("expected error for non-member")
	}
}

func TestService_ValidateTeamMembership(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "team1", "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.AddMember(ctx, team.ID, "agent-1", "browser_specialist"); err != nil {
		t.Fatal(err)
	}

	ok, err := svc.ValidateTeamMembership(ctx, team.ID, "agent-1")
	if err != nil {
		t.Fatalf("ValidateTeamMembership: %v", err)
	}
	if !ok {
		t.Fatal("expected membership to be valid")
	}

	ok, err = svc.ValidateTeamMembership(ctx, team.ID, "stranger")
	if err != nil {
		t.Fatalf("ValidateTeamMembership stranger: %v", err)
	}
	if ok {
		t.Fatal("expected membership to be invalid for stranger")
	}
}

func TestService_ValidateTeamMembership_TeamNotFound(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.ValidateTeamMembership(ctx, "ghost-team", "agent-1")
	if err == nil {
		t.Fatal("expected error for nonexistent team")
	}
}

func TestService_RemoveMember_AutoDissolve(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	team, err := svc.CreateTeam(ctx, "team1", "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.AddMember(ctx, team.ID, "agent-1", "browser_specialist"); err != nil {
		t.Fatal(err)
	}

	if err := svc.RemoveMember(ctx, team.ID, "agent-1"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	got, err := svc.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LifecycleState != LifecycleDissolved {
		t.Fatalf("expected auto-dissolved team, got %q", got.LifecycleState)
	}
}

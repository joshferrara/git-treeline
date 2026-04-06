package tui

import (
	"testing"

	"github.com/git-treeline/git-treeline/internal/registry"
)

// --- buildFlatList ---

func testSnapshot() Snapshot {
	return Snapshot{
		Projects: []string{"api", "frontend"},
		Worktrees: []WorktreeStatus{
			{Project: "api", Branch: "main", WorktreeName: "api-main", Ports: []int{3000}},
			{Project: "api", Branch: "feature-x", WorktreeName: "api-feature-x", Ports: []int{3010}},
			{Project: "frontend", Branch: "main", WorktreeName: "fe-main", Ports: []int{3020}},
			{Project: "frontend", Branch: "redesign", WorktreeName: "fe-redesign", Ports: []int{3030}},
		},
	}
}

func TestBuildFlatList_NoFilter(t *testing.T) {
	snap := testSnapshot()
	entries := buildFlatList(snap, "")

	// 2 project headers + 4 worktree rows = 6
	if len(entries) != 6 {
		t.Fatalf("expected 6 entries, got %d", len(entries))
	}

	if !entries[0].projectHeader || entries[0].project != "api" {
		t.Errorf("entry 0: expected api header, got %+v", entries[0])
	}
	if entries[1].projectHeader || entries[1].wt.Branch != "main" {
		t.Errorf("entry 1: expected api/main worktree, got %+v", entries[1])
	}
	if !entries[3].projectHeader || entries[3].project != "frontend" {
		t.Errorf("entry 3: expected frontend header, got %+v", entries[3])
	}
}

func TestBuildFlatList_WithFilter(t *testing.T) {
	snap := testSnapshot()
	entries := buildFlatList(snap, "redesign")

	// Only frontend/redesign matches, so: 1 header + 1 worktree = 2
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if !entries[0].projectHeader || entries[0].project != "frontend" {
		t.Errorf("entry 0: expected frontend header, got %+v", entries[0])
	}
	if entries[1].wt.Branch != "redesign" {
		t.Errorf("entry 1: expected redesign worktree, got branch %s", entries[1].wt.Branch)
	}
}

func TestBuildFlatList_FilterMatchesNothing(t *testing.T) {
	snap := testSnapshot()
	entries := buildFlatList(snap, "zzzzz")

	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestBuildFlatList_FilterCaseInsensitive(t *testing.T) {
	snap := testSnapshot()
	entries := buildFlatList(snap, "FEATURE")

	// api/feature-x matches
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (header + worktree), got %d", len(entries))
	}
	if entries[1].wt.Branch != "feature-x" {
		t.Errorf("expected feature-x, got %s", entries[1].wt.Branch)
	}
}

// --- matchesFilter ---

func TestMatchesFilter_Project(t *testing.T) {
	wt := &WorktreeStatus{Project: "MyApp", Branch: "main", WorktreeName: "myapp-main"}
	if !matchesFilter(wt, "myapp") {
		t.Error("expected filter 'myapp' to match project 'MyApp'")
	}
}

func TestMatchesFilter_Branch(t *testing.T) {
	wt := &WorktreeStatus{Project: "api", Branch: "feature-auth", WorktreeName: "api-auth"}
	if !matchesFilter(wt, "auth") {
		t.Error("expected filter 'auth' to match branch 'feature-auth'")
	}
}

func TestMatchesFilter_NoMatch(t *testing.T) {
	wt := &WorktreeStatus{Project: "api", Branch: "main", WorktreeName: "api-main"}
	if matchesFilter(wt, "zzz") {
		t.Error("expected filter 'zzz' to not match")
	}
}

// --- moveCursor ---

func buildTestModel() Model {
	snap := testSnapshot()
	m := Model{
		snapshot: snap,
		flatList: buildFlatList(snap, ""),
		height:   40,
		width:    120,
	}
	m.clampCursor()
	return m
}

func TestMoveCursor_Down(t *testing.T) {
	m := buildTestModel()
	// Should start at index 1 (first non-header)
	if m.cursor != 1 {
		t.Fatalf("expected initial cursor at 1, got %d", m.cursor)
	}

	m.moveCursor(1)
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 after moving down, got %d", m.cursor)
	}
}

func TestMoveCursor_DownSkipsHeader(t *testing.T) {
	m := buildTestModel()
	m.cursor = 2 // api/feature-x, next is frontend header at 3

	m.moveCursor(1)
	// Should skip header at 3, land on 4 (frontend/main)
	if m.cursor != 4 {
		t.Errorf("expected cursor at 4 (skip header), got %d", m.cursor)
	}
}

func TestMoveCursor_UpSkipsHeader(t *testing.T) {
	m := buildTestModel()
	m.cursor = 4 // frontend/main, prev is header at 3

	m.moveCursor(-1)
	// Should skip header at 3, land on 2 (api/feature-x)
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 (skip header), got %d", m.cursor)
	}
}

func TestMoveCursor_UpAtTopStays(t *testing.T) {
	m := buildTestModel()
	m.cursor = 1 // first selectable row

	m.moveCursor(-1)
	// Entry 0 is a header, no valid entry above — should stay at 1
	if m.cursor != 1 {
		t.Errorf("expected cursor to stay at 1 at top boundary, got %d", m.cursor)
	}
}

func TestMoveCursor_DownAtBottomStays(t *testing.T) {
	m := buildTestModel()
	last := len(m.flatList) - 1
	m.cursor = last

	m.moveCursor(1)
	if m.cursor != last {
		t.Errorf("expected cursor to stay at %d at bottom boundary, got %d", last, m.cursor)
	}
}

// --- clampCursor ---

func TestClampCursor_NegativeBecomesFirstSelectable(t *testing.T) {
	m := buildTestModel()
	m.cursor = -5
	m.clampCursor()

	if m.cursor < 0 || m.flatList[m.cursor].projectHeader {
		t.Errorf("expected cursor on a selectable row, got %d", m.cursor)
	}
}

func TestClampCursor_BeyondEndClamps(t *testing.T) {
	m := buildTestModel()
	m.cursor = 999
	m.clampCursor()

	if m.cursor >= len(m.flatList) {
		t.Errorf("expected cursor within bounds, got %d (len=%d)", m.cursor, len(m.flatList))
	}
}

// --- renderedLinesBetween ---

func TestRenderedLinesBetween(t *testing.T) {
	m := buildTestModel()
	// entries: [header, wt, wt, header, wt, wt]
	// lines:     2      1   1    2      1   1 = 8 total

	lines := m.renderedLinesBetween(0, 5)
	if lines != 8 {
		t.Errorf("expected 8 rendered lines for full list, got %d", lines)
	}

	// Just the first project group: header(2) + 2 worktrees(2) = 4
	lines = m.renderedLinesBetween(0, 2)
	if lines != 4 {
		t.Errorf("expected 4 lines for entries 0-2, got %d", lines)
	}

	// Single worktree row
	lines = m.renderedLinesBetween(1, 1)
	if lines != 1 {
		t.Errorf("expected 1 line for single worktree, got %d", lines)
	}

	// Single header
	lines = m.renderedLinesBetween(0, 0)
	if lines != 2 {
		t.Errorf("expected 2 lines for single header, got %d", lines)
	}
}

// --- ensureCursorVisible ---

func TestEnsureCursorVisible_ScrollsDown(t *testing.T) {
	m := buildTestModel()
	m.height = 8 // very short — listVisibleLines = 8-2-2-1 = 3
	m.cursor = 5 // last entry
	m.scrollOffset = 0

	m.ensureCursorVisible()

	if m.scrollOffset == 0 {
		t.Error("expected scrollOffset to increase when cursor is below visible area")
	}
}

func TestEnsureCursorVisible_ScrollsUp(t *testing.T) {
	m := buildTestModel()
	m.height = 20 // enough room for visible lines > margin
	m.scrollOffset = 4
	m.cursor = 1

	m.ensureCursorVisible()

	if m.scrollOffset > m.cursor {
		t.Errorf("expected scrollOffset <= cursor, got offset=%d cursor=%d", m.scrollOffset, m.cursor)
	}
}

// --- extractLinks ---

func TestExtractLinks_WithLinks(t *testing.T) {
	a := registry.Allocation{
		"worktree": "/tmp/test",
		"links": map[string]any{
			"api":   "http://localhost:3040",
			"redis": "redis://localhost:6380",
		},
	}

	links := extractLinks(a)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links["api"] != "http://localhost:3040" {
		t.Errorf("expected api link, got %s", links["api"])
	}
}

func TestExtractLinks_NoLinks(t *testing.T) {
	a := registry.Allocation{"worktree": "/tmp/test"}
	links := extractLinks(a)
	if links != nil {
		t.Errorf("expected nil links, got %v", links)
	}
}

func TestExtractLinks_EmptyLinksMap(t *testing.T) {
	a := registry.Allocation{
		"worktree": "/tmp/test",
		"links":    map[string]any{},
	}
	links := extractLinks(a)
	if links != nil {
		t.Errorf("expected nil for empty links map, got %v", links)
	}
}

func TestExtractLinks_NonStringValues(t *testing.T) {
	a := registry.Allocation{
		"links": map[string]any{
			"api":   42,
			"redis": "redis://localhost",
		},
	}
	links := extractLinks(a)
	if len(links) != 1 {
		t.Fatalf("expected 1 link (non-string filtered), got %d", len(links))
	}
	if links["redis"] != "redis://localhost" {
		t.Errorf("expected redis link, got %s", links["redis"])
	}
}

// --- selectedWorktree ---

func TestSelectedWorktree_ValidCursor(t *testing.T) {
	m := buildTestModel()
	m.cursor = 1
	wt := m.selectedWorktree()
	if wt == nil {
		t.Fatal("expected non-nil worktree")
	}
	if wt.Branch != "main" || wt.Project != "api" {
		t.Errorf("expected api/main, got %s/%s", wt.Project, wt.Branch)
	}
}

func TestSelectedWorktree_OnHeader(t *testing.T) {
	m := buildTestModel()
	m.cursor = 0 // header
	wt := m.selectedWorktree()
	if wt != nil {
		t.Errorf("expected nil when cursor is on header, got %+v", wt)
	}
}

func TestSelectedWorktree_OutOfBounds(t *testing.T) {
	m := buildTestModel()
	m.cursor = -1
	if m.selectedWorktree() != nil {
		t.Error("expected nil for negative cursor")
	}
	m.cursor = 999
	if m.selectedWorktree() != nil {
		t.Error("expected nil for out-of-bounds cursor")
	}
}

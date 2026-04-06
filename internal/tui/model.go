package tui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
)

const pollInterval = 2 * time.Second

// focusPane tracks which panel has keyboard focus.
type focusPane int

const (
	paneList focusPane = iota
	paneDetail
)

// Model is the root Bubble Tea model for the gtl dashboard.
type Model struct {
	snapshot     Snapshot
	width        int
	height       int
	focus        focusPane
	cursor       int // selected index into flatList
	scrollOffset int // first visible index in the list panel
	flatList     []flatEntry
	ready        bool
	polling      bool
	spinner      spinner.Model
	filterMode   bool
	filterText   string
	showHelp     bool
	confirmKind  string // "release" or ""
	quitting     bool
}

// flatEntry is a denormalized row in the worktree list.
// projectHeader=true means this row is a group header, not a selectable worktree.
type flatEntry struct {
	projectHeader bool
	project       string
	wt            *WorktreeStatus
}

type tickMsg time.Time
type dataMsg Snapshot

func doTick() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func doPoll() tea.Cmd {
	return func() tea.Msg {
		return dataMsg(Poll())
	}
}

// NewModel creates the initial dashboard model.
func NewModel() Model {
	s := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(highlight)),
	)
	return Model{spinner: s, polling: true}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(doPoll(), doTick(), m.spinner.Tick)
}

// buildFlatList converts the snapshot into a flat list of project headers + worktree rows.
func buildFlatList(snap Snapshot, filter string) []flatEntry {
	grouped := make(map[string][]WorktreeStatus, len(snap.Projects))
	for i := range snap.Worktrees {
		wt := &snap.Worktrees[i]
		if filter != "" && !matchesFilter(wt, filter) {
			continue
		}
		grouped[wt.Project] = append(grouped[wt.Project], *wt)
	}

	var entries []flatEntry
	for _, proj := range snap.Projects {
		wts := grouped[proj]
		if len(wts) == 0 {
			continue
		}
		entries = append(entries, flatEntry{projectHeader: true, project: proj})
		for i := range wts {
			entries = append(entries, flatEntry{project: proj, wt: &wts[i]})
		}
	}
	return entries
}

func matchesFilter(wt *WorktreeStatus, filter string) bool {
	f := strings.ToLower(filter)
	return strings.Contains(strings.ToLower(wt.Project), f) ||
		strings.Contains(strings.ToLower(wt.Branch), f) ||
		strings.Contains(strings.ToLower(wt.WorktreeName), f)
}

// selectedWorktree returns the WorktreeStatus at the cursor, or nil if on a header.
func (m *Model) selectedWorktree() *WorktreeStatus {
	if m.cursor < 0 || m.cursor >= len(m.flatList) {
		return nil
	}
	return m.flatList[m.cursor].wt
}

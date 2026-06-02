package bookmarksview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/natevick/stui/internal/bookmarks"
)

// Item represents a bookmark in the list
type Item struct {
	bookmark bookmarks.Bookmark
}

func (i Item) Title() string       { return "🔖 " + i.bookmark.DisplayName() }
func (i Item) Description() string { return i.bookmark.Path() }
func (i Item) FilterValue() string { return i.bookmark.DisplayName() }

// Action represents an action to take
type Action int

const (
	ActionNone Action = iota
	ActionSelect
	ActionDelete
)

// Model is the bookmarks view model
type Model struct {
	list       list.Model
	bookmarks  []bookmarks.Bookmark
	store      *bookmarks.Store
	err        error
	width      int
	height     int
	action     Action
	selectedID string
}

// New creates a new bookmarks view
func New() Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("213")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("213"))

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Bookmarks"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("213")).
		Padding(0, 1)

	return Model{
		list: l,
	}
}

// SetSize sets the view size
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height)
}

// IsFiltering reports whether a filter is active (being typed or applied), so
// callers can let the list handle Esc (cancel/clear filter) instead of quitting.
func (m Model) IsFiltering() bool {
	return m.list.FilterState() != list.Unfiltered
}

// SetStore sets the bookmark store
func (m *Model) SetStore(store *bookmarks.Store) {
	m.store = store
	m.Refresh()
}

// Refresh reloads bookmarks from store
func (m *Model) Refresh() {
	if m.store == nil {
		return
	}

	m.bookmarks = m.store.List()
	items := make([]list.Item, len(m.bookmarks))
	for i, b := range m.bookmarks {
		items[i] = Item{bookmark: b}
	}
	m.list.SetItems(items)
}

// SetError sets an error state
func (m *Model) SetError(err error) {
	m.err = err
}

// SelectedBookmark returns the currently selected bookmark
func (m Model) SelectedBookmark() (bookmarks.Bookmark, bool) {
	if item, ok := m.list.SelectedItem().(Item); ok {
		return item.bookmark, true
	}
	return bookmarks.Bookmark{}, false
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	m.action = ActionNone

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys if filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if item, ok := m.list.SelectedItem().(Item); ok {
				m.action = ActionSelect
				m.selectedID = item.bookmark.ID
				return m, nil
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("x", "delete"))):
			if item, ok := m.list.SelectedItem().(Item); ok {
				m.action = ActionDelete
				m.selectedID = item.bookmark.ID
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the view
func (m Model) View() string {
	if m.store == nil {
		return m.renderLoading()
	}

	if m.err != nil {
		return m.renderError()
	}

	if len(m.bookmarks) == 0 {
		return m.renderEmpty()
	}

	return m.list.View()
}

func (m Model) renderLoading() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render("Loading bookmarks...")
}

func (m Model) renderEmpty() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(lipgloss.Color("240"))

	var sb strings.Builder
	sb.WriteString("No bookmarks yet\n\n")
	sb.WriteString("Navigate to a location and press 'b' to bookmark it")

	return style.Render(sb.String())
}

func (m Model) renderError() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(lipgloss.Color("196"))

	return style.Render(fmt.Sprintf("Error: %v", m.err))
}

// Action returns the pending action
func (m Model) Action() Action {
	return m.action
}

// ConsumeAction clears and returns the action
func (m *Model) ConsumeAction() (Action, string) {
	action := m.action
	id := m.selectedID
	m.action = ActionNone
	m.selectedID = ""
	return action, id
}

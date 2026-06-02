package profiles

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/natevick/stui/internal/providers"
)

// Item represents a provider entry in the list
type Item struct {
	entry providers.Entry
}

func (i Item) Title() string {
	return fmt.Sprintf("%s  [%s]", i.entry.Name, i.entry.Provider)
}
func (i Item) Description() string {
	parts := []string{}
	if i.entry.Region != "" {
		parts = append(parts, "Region: "+i.entry.Region)
	}
	if i.entry.AccountID != "" {
		parts = append(parts, "Account: "+i.entry.AccountID)
	}
	if i.entry.Endpoint != "" {
		parts = append(parts, "Endpoint: "+i.entry.Endpoint)
	}
	if len(parts) == 0 {
		return i.entry.Provider
	}
	return strings.Join(parts, " | ")
}

// FilterValue lets the user filter by both name and provider.
func (i Item) FilterValue() string { return i.entry.Name + " " + i.entry.Provider }

// SelectedMsg is sent when a profile/alias is selected
type SelectedMsg struct {
	Profile  string
	Provider string
}

// Model is the profile picker view model
type Model struct {
	list     list.Model
	entries  []providers.Entry
	width    int
	height   int
	selected string
}

// New creates a new profile picker view
func New() Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("39")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("39"))

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Select Profile / Alias"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
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

// LoadEntries loads selectable entries from all providers (aws profiles, mc
// aliases, stui endpoints). Healthy providers are always loaded; a non-nil
// error reports providers that failed to load (e.g. a malformed config) without
// hiding the ones that worked.
func (m *Model) LoadEntries() error {
	entries, err := providers.List()
	m.entries = entries
	items := make([]list.Item, len(m.entries))
	for i, e := range m.entries {
		items[i] = Item{entry: e}
	}
	m.list.SetItems(items)
	return err
}

// SelectedProfile returns the selected profile name
func (m *Model) SelectedProfile() string {
	return m.selected
}

// ClearSelection clears the selection
func (m *Model) ClearSelection() {
	m.selected = ""
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys if filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			if item, ok := m.list.SelectedItem().(Item); ok {
				m.selected = item.entry.Name
				entry := item.entry
				return m, func() tea.Msg {
					return SelectedMsg{Profile: entry.Name, Provider: entry.Provider}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the view
func (m Model) View() string {
	if len(m.entries) == 0 {
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("196"))

		return style.Render("No profiles or aliases found.\n\nConfigure an AWS profile (~/.aws/config), an mc alias (~/.mc/config.json),\nor a stui endpoint (~/.config/stui/config.json).")
	}

	return m.list.View()
}

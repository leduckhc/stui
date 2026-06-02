package profiles

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/natevick/stui/internal/aws"
)

// Item represents a profile in the list
type Item struct {
	profile aws.ProfileInfo
}

func (i Item) Title() string { return i.profile.Name }
func (i Item) Description() string {
	desc := fmt.Sprintf("Region: %s", i.profile.Region)
	if i.profile.AccountID != "" {
		desc += fmt.Sprintf(" | Account: %s", i.profile.AccountID)
	}
	return desc
}
func (i Item) FilterValue() string { return i.profile.Name }

// SelectedMsg is sent when a profile is selected
type SelectedMsg struct {
	Profile string
}

// Model is the profile picker view model
type Model struct {
	list     list.Model
	profiles []aws.ProfileInfo
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
	l.Title = "Select AWS Profile"
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

// LoadProfiles loads available AWS profiles
func (m *Model) LoadProfiles() error {
	profiles, err := aws.ListProfiles()
	if err != nil {
		return err
	}

	m.profiles = profiles
	items := make([]list.Item, len(profiles))
	for i, p := range profiles {
		items[i] = Item{profile: p}
	}
	m.list.SetItems(items)
	return nil
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
				m.selected = item.profile.Name
				return m, func() tea.Msg {
					return SelectedMsg{Profile: item.profile.Name}
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
	if len(m.profiles) == 0 {
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("196"))

		return style.Render("No AWS SSO profiles found in ~/.aws/config\n\nRun 'aws configure sso' to set up a profile")
	}

	return m.list.View()
}

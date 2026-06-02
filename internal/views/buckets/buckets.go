package buckets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/natevick/stui/internal/aws"
)

// Item represents a bucket in the list
type Item struct {
	bucket aws.Bucket
}

func (i Item) Title() string       { return i.bucket.Name }
func (i Item) Description() string { return fmt.Sprintf("Created: %s", i.bucket.CreationDate.Format("2006-01-02")) }
func (i Item) FilterValue() string { return i.bucket.Name }

// Action represents an action to take
type Action int

const (
	ActionNone Action = iota
	ActionSelect
	ActionBookmark
)

// Model is the buckets view model
type Model struct {
	list           list.Model
	buckets        []aws.Bucket
	loading        bool
	err            error
	width          int
	height         int
	selected       string
	action         Action
	selectedBucket string
}

// New creates a new buckets view
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
	l.Title = "S3 Buckets"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Padding(0, 1)

	return Model{
		list:    l,
		loading: true,
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

// SetBuckets updates the bucket list
func (m *Model) SetBuckets(buckets []aws.Bucket) {
	m.buckets = buckets
	m.loading = false

	items := make([]list.Item, len(buckets))
	for i, b := range buckets {
		items[i] = Item{bucket: b}
	}
	m.list.SetItems(items)
}

// SetError sets an error state
func (m *Model) SetError(err error) {
	m.err = err
	m.loading = false
}

// SetLoading sets the loading state
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

// SelectedBucket returns the currently selected bucket name
func (m *Model) SelectedBucket() string {
	if item, ok := m.list.SelectedItem().(Item); ok {
		return item.bucket.Name
	}
	return ""
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
				m.selectedBucket = item.bucket.Name
				m.action = ActionSelect
				return m, nil
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("b"))):
			if item, ok := m.list.SelectedItem().(Item); ok {
				m.selectedBucket = item.bucket.Name
				m.action = ActionBookmark
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
	if m.loading {
		return m.renderLoading()
	}

	if m.err != nil {
		return m.renderError()
	}

	return m.list.View()
}

func (m Model) renderLoading() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render("Loading buckets...")
}

func (m Model) renderError() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(lipgloss.Color("196"))

	var sb strings.Builder
	sb.WriteString("Error loading buckets:\n\n")
	sb.WriteString(m.err.Error())
	sb.WriteString("\n\nMake sure you have run: aws sso login --profile <profile>")

	return style.Render(sb.String())
}

// HasSelection returns true if a bucket was selected
func (m Model) HasSelection() bool {
	return m.selected != ""
}

// ConsumeSelection returns and clears the selection
func (m *Model) ConsumeSelection() string {
	s := m.selected
	m.selected = ""
	return s
}

// ConsumeAction clears and returns the action
func (m *Model) ConsumeAction() (Action, string) {
	action := m.action
	bucket := m.selectedBucket
	m.action = ActionNone
	m.selectedBucket = ""
	return action, bucket
}

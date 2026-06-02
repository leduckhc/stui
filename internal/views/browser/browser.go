package browser

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/natevick/stui/internal/aws"
)

// Item represents an S3 object in the list
type Item struct {
	object   aws.S3Object
	selected bool
}

func (i Item) Title() string {
	name := i.object.DisplayName()
	var icon string
	if i.selected {
		icon = "✓ "
	} else {
		icon = "  "
	}
	if i.object.IsPrefix {
		return icon + "📁 " + name
	}
	return icon + "📄 " + name
}

func (i Item) Description() string {
	if i.object.IsPrefix {
		return "folder"
	}
	return fmt.Sprintf("%s  •  %s",
		humanize.Bytes(uint64(i.object.Size)),
		i.object.LastModified.Format("2006-01-02 15:04"),
	)
}

func (i Item) FilterValue() string {
	return i.object.DisplayName()
}

// Action represents an action to take
type Action int

const (
	ActionNone Action = iota
	ActionNavigate
	ActionBack
	ActionDownload
	ActionSync
	ActionBookmark
)

// Model is the browser view model
type Model struct {
	list    list.Model
	bucket  string
	prefix  string
	history []string // prefix history for back navigation
	objects []aws.S3Object
	loading bool
	err     error
	width   int
	height  int

	// cursorByPrefix remembers the cursor position for each visited prefix so
	// navigating back (or refreshing) restores where the user was.
	cursorByPrefix map[string]int

	// Multi-select
	selected map[string]bool // map of Key -> selected

	// Pending action
	action          Action
	selectedObject  aws.S3Object
	selectedObjects []aws.S3Object // for multi-select downloads
}

// New creates a new browser view
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
	l.Title = "Objects"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Padding(0, 1)

	return Model{
		list:           l,
		history:        []string{},
		selected:       make(map[string]bool),
		cursorByPrefix: make(map[string]int),
	}
}

// SetSize sets the view size
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height-2) // Reserve space for path
}

// IsFiltering reports whether a filter is active (being typed or applied), so
// callers can let the list handle Esc (cancel/clear filter) instead of quitting.
func (m Model) IsFiltering() bool {
	return m.list.FilterState() != list.Unfiltered
}

// RememberCursor saves the current cursor position for the current prefix so a
// subsequent reload (e.g. refresh) can restore it.
func (m *Model) RememberCursor() {
	m.cursorByPrefix[m.prefix] = m.list.Index()
}

// SetBucket sets the current bucket
func (m *Model) SetBucket(bucket string) {
	m.bucket = bucket
	m.prefix = ""
	m.history = []string{}
	m.selected = make(map[string]bool) // Clear selection
	m.cursorByPrefix = make(map[string]int)
	m.updateTitle()
}

// SetPrefix sets the current prefix
func (m *Model) SetPrefix(prefix string) {
	m.prefix = prefix
	m.updateTitle()
}

// SetObjects updates the object list
func (m *Model) SetObjects(objects []aws.S3Object) {
	m.objects = objects
	m.loading = false
	m.selected = make(map[string]bool) // Clear selection when navigating

	items := make([]list.Item, len(objects))
	for i, obj := range objects {
		items[i] = Item{object: obj, selected: false}
	}
	m.list.SetItems(items)

	// Restore the remembered cursor position for this prefix.
	if idx, ok := m.cursorByPrefix[m.prefix]; ok && len(objects) > 0 {
		if idx >= len(objects) {
			idx = len(objects) - 1
		}
		if idx < 0 {
			idx = 0
		}
		m.list.Select(idx)
	}
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

// Bucket returns the current bucket
func (m Model) Bucket() string {
	return m.bucket
}

// Prefix returns the current prefix
func (m Model) Prefix() string {
	return m.prefix
}

// SelectedObject returns the currently selected object
func (m Model) SelectedObject() (aws.S3Object, bool) {
	if item, ok := m.list.SelectedItem().(Item); ok {
		return item.object, true
	}
	return aws.S3Object{}, false
}

func (m *Model) updateTitle() {
	if m.bucket == "" {
		m.list.Title = "Objects"
		return
	}
	path := fmt.Sprintf("s3://%s/%s", m.bucket, m.prefix)
	m.list.Title = path
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
		case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
			// Toggle selection with spacebar
			if item, ok := m.list.SelectedItem().(Item); ok {
				m.toggleSelection(item.object.Key)
				m.refreshListItems()
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "right"))):
			if item, ok := m.list.SelectedItem().(Item); ok {
				if item.object.IsPrefix {
					// Remember where we are, then navigate into the prefix
					m.cursorByPrefix[m.prefix] = m.list.Index()
					m.history = append(m.history, m.prefix)
					m.prefix = item.object.Key
					m.selectedObject = item.object
					m.action = ActionNavigate
					m.updateTitle()
					return m, nil
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("backspace", "left"))):
			if len(m.history) > 0 {
				m.cursorByPrefix[m.prefix] = m.list.Index()
				m.prefix = m.history[len(m.history)-1]
				m.history = m.history[:len(m.history)-1]
				m.action = ActionBack
				m.updateTitle()
				return m, nil
			} else if m.prefix != "" {
				// Go back to bucket root
				m.cursorByPrefix[m.prefix] = m.list.Index()
				m.prefix = ""
				m.action = ActionBack
				m.updateTitle()
				return m, nil
			}
			// At bucket root with no history: swallow so it doesn't fall
			// through (the global handler already declined to switch tabs).
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
			// Download selected items, or current item if none selected
			selectedObjs := m.GetSelectedObjects()
			if len(selectedObjs) > 0 {
				m.selectedObjects = selectedObjs
				m.action = ActionDownload
			} else if item, ok := m.list.SelectedItem().(Item); ok {
				m.selectedObject = item.object
				m.action = ActionDownload
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
			m.action = ActionSync
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("b"))):
			m.action = ActionBookmark
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// toggleSelection toggles the selection state of an object
func (m *Model) toggleSelection(key string) {
	if m.selected[key] {
		delete(m.selected, key)
	} else {
		m.selected[key] = true
	}
}

// refreshListItems updates the list items with current selection state
func (m *Model) refreshListItems() {
	idx := m.list.Index()
	items := make([]list.Item, len(m.objects))
	for i, obj := range m.objects {
		items[i] = Item{object: obj, selected: m.selected[obj.Key]}
	}
	m.list.SetItems(items)
	m.list.Select(idx) // Preserve cursor position
}

// GetSelectedObjects returns all selected objects
func (m Model) GetSelectedObjects() []aws.S3Object {
	var objs []aws.S3Object
	for _, obj := range m.objects {
		if m.selected[obj.Key] {
			objs = append(objs, obj)
		}
	}
	return objs
}

// SelectionCount returns the number of selected items
func (m Model) SelectionCount() int {
	return len(m.selected)
}

// ClearSelection clears all selections
func (m *Model) ClearSelection() {
	m.selected = make(map[string]bool)
	m.refreshListItems()
}

// View renders the view
func (m Model) View() string {
	if m.bucket == "" {
		return m.renderNoBucket()
	}

	if m.loading {
		return m.renderLoading()
	}

	if m.err != nil {
		return m.renderError()
	}

	var sb strings.Builder

	// Path breadcrumb
	path := m.renderPath()
	sb.WriteString(path)
	sb.WriteString("\n\n")

	// List
	sb.WriteString(m.list.View())

	return sb.String()
}

func (m Model) renderPath() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	var path string
	if m.prefix == "" {
		path = fmt.Sprintf("📦 %s", m.bucket)
	} else {
		// Build breadcrumb
		parts := strings.Split(strings.TrimSuffix(m.prefix, "/"), "/")
		var breadcrumbs []string
		breadcrumbs = append(breadcrumbs, "📦 "+m.bucket)
		for _, part := range parts {
			if part != "" {
				breadcrumbs = append(breadcrumbs, part)
			}
		}
		path = strings.Join(breadcrumbs, " / ")
	}

	// Show selection count
	if count := len(m.selected); count > 0 {
		selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
		path += selStyle.Render(fmt.Sprintf("  [%d selected]", count))
	}

	return style.Render(path)
}

func (m Model) renderNoBucket() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(lipgloss.Color("240"))

	return style.Render("Select a bucket from the Buckets view (press 1)")
}

func (m Model) renderLoading() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render("Loading objects...")
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
func (m *Model) ConsumeAction() (Action, aws.S3Object, []aws.S3Object) {
	action := m.action
	obj := m.selectedObject
	objs := m.selectedObjects
	m.action = ActionNone
	m.selectedObject = aws.S3Object{}
	m.selectedObjects = nil
	return action, obj, objs
}

// DefaultDownloadPath returns a sensible default download path
func (m Model) DefaultDownloadPath(obj aws.S3Object) string {
	if obj.IsPrefix {
		// For prefix, use the folder name
		name := strings.TrimSuffix(obj.Key, "/")
		parts := strings.Split(name, "/")
		if len(parts) > 0 {
			return "./" + parts[len(parts)-1]
		}
		return "./download"
	}
	// For file, use the filename
	return "./" + filepath.Base(obj.Key)
}

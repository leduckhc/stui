package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all key bindings for the application
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Back     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// View switching
	Tab         key.Binding
	ShiftTab    key.Binding
	Buckets     key.Binding
	Browser     key.Binding
	Bookmarks   key.Binding

	// Actions
	Select      key.Binding
	Download    key.Binding
	Sync        key.Binding
	AddBookmark key.Binding
	Delete      key.Binding
	Refresh     key.Binding
	Cancel      key.Binding

	// App
	Help key.Binding
	Quit key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "back/prev tab"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "open/next tab"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "back"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "go to bottom"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev tab"),
		),
		Buckets: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "buckets"),
		),
		Browser: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "browser"),
		),
		Bookmarks: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "bookmarks"),
		),
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "select"),
		),
		Download: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "download"),
		),
		Sync: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sync"),
		),
		AddBookmark: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "add bookmark"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel/quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns keybindings for the short help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Cancel, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Right, k.Left},
		{k.Tab, k.Buckets, k.Browser, k.Bookmarks},
		{k.Download, k.Sync, k.AddBookmark, k.Refresh},
		{k.Help, k.Cancel, k.Quit},
	}
}

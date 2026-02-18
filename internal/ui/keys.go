package ui

import "github.com/charmbracelet/bubbles/key"

// SearchKeyMap defines key bindings for the search mode.
type SearchKeyMap struct {
	Run       key.Binding
	Toggle    key.Binding
	SelectAll key.Binding
	Up        key.Binding
	Down      key.Binding
	Quit      key.Binding
}

var searchKeys = SearchKeyMap{
	Run: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "run"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "select"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("ctrl+a"),
		key.WithHelp("Ctrl+A", "select all"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "ctrl+p", "ctrl+k"),
		key.WithHelp("↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "ctrl+n", "ctrl+j"),
		key.WithHelp("↓", "down"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("Ctrl+C", "quit"),
	),
}

// RunningKeyMap defines key bindings for the running mode.
type RunningKeyMap struct {
	Cancel key.Binding
	Quit   key.Binding
}

var runningKeys = RunningKeyMap{
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "cancel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("Ctrl+C", "quit"),
	),
}

// ResultsKeyMap defines key bindings for the results mode.
type ResultsKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Back     key.Binding
	Rerun    key.Binding
	RerunAll key.Binding
	Filter   key.Binding
	Open     key.Binding
	Quit     key.Binding
}

var resultsKeys = ResultsKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k/↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j/↓", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "back"),
	),
	Right: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "detail"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "back to search"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "search"),
	),
	Rerun: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "rerun"),
	),
	RerunAll: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "rerun all"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "fails only"),
	),
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in editor"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

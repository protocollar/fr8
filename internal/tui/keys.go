package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Back    key.Binding
	Archive key.Binding
	Shell   key.Binding
	Run     key.Binding
	Browser key.Binding
	Quit    key.Binding
	Yes     key.Binding
	No      key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "move down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Archive: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "archive"),
	),
	Shell: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "shell"),
	),
	Run: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "run"),
	),
	Browser: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "browser"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Yes: key.NewBinding(
		key.WithKeys("y"),
	),
	No: key.NewBinding(
		key.WithKeys("n", "esc"),
	),
}

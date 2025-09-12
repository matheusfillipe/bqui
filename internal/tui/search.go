package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type SearchModel struct {
	input string
}

func NewSearchModel() SearchModel {
	return SearchModel{
		input: "",
	}
}

func (m SearchModel) Init() tea.Cmd {
	return nil
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case "ctrl+u":
			m.input = ""
		default:
			if len(msg.String()) == 1 {
				m.input += msg.String()
			}
		}
	}
	return m, nil
}

func (m SearchModel) View() string {
	return m.input
}

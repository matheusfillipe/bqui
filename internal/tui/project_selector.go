package tui

import (
	"fmt"
	"strings"

	"bqui/internal/bigquery"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type ProjectSelectorModel struct {
	projects []*bigquery.Project
	cursor   int
	filter   string
}

func NewProjectSelectorModel() ProjectSelectorModel {
	return ProjectSelectorModel{
		projects: make([]*bigquery.Project, 0),
		cursor:   0,
		filter:   "",
	}
}

func (m ProjectSelectorModel) Init() tea.Cmd {
	return nil
}

func (m ProjectSelectorModel) Update(msg tea.Msg) (ProjectSelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeypress(msg)
	case ProjectsLoadedMsg:
		m.projects = msg.Projects
		return m, nil
	}
	return m, nil
}

func (m ProjectSelectorModel) handleKeypress(msg tea.KeyMsg) (ProjectSelectorModel, tea.Cmd) {
	filteredProjects := m.getFilteredProjects()
	
	switch {
	case key.Matches(msg, DefaultKeyMap().Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, DefaultKeyMap().Down):
		if m.cursor < len(filteredProjects)-1 {
			m.cursor++
		}

	case key.Matches(msg, DefaultKeyMap().Enter):
		filteredProjects := m.getFilteredProjects()
		if len(filteredProjects) > 0 && m.cursor < len(filteredProjects) {
			selectedProject := filteredProjects[m.cursor]
			return m, func() tea.Msg {
				return ProjectSelectedMsg{Project: selectedProject}
			}
		}
		return m, nil

	case key.Matches(msg, DefaultKeyMap().Search):
		return m, nil
	}

	return m, nil
}

func (m ProjectSelectorModel) getFilteredProjects() []*bigquery.Project {
	if m.filter == "" {
		return m.projects
	}

	var filtered []*bigquery.Project
	for _, project := range m.projects {
		if strings.Contains(strings.ToLower(project.ID), strings.ToLower(m.filter)) ||
		   strings.Contains(strings.ToLower(project.Name), strings.ToLower(m.filter)) {
			filtered = append(filtered, project)
		}
	}
	return filtered
}

func (m ProjectSelectorModel) View() string {
	var content strings.Builder

	content.WriteString(HeaderStyle.Render("🚀 Select Project") + "\n\n")

	if m.filter != "" {
		content.WriteString(SubtleItemStyle.Render("Filter: "+m.filter) + "\n\n")
	}

	filteredProjects := m.getFilteredProjects()

	if len(filteredProjects) == 0 {
		content.WriteString(SubtleItemStyle.Render("No projects found"))
		return content.String()
	}

	for i, project := range filteredProjects {
		style := ItemStyle
		if i == m.cursor {
			style = SelectedItemStyle
		}

		projectDisplay := project.ID
		if project.Name != project.ID {
			projectDisplay += " (" + project.Name + ")"
		}

		content.WriteString(style.Render("  📦 "+projectDisplay) + "\n")
	}

	content.WriteString("\n" + SubtleItemStyle.Render("Projects: ") + SubtleItemStyle.Render(fmt.Sprintf("%d", len(filteredProjects))))
	content.WriteString("\n" + HelpStyle.Render("Use ↑/↓ to navigate, Enter to select, Esc to cancel"))

	return content.String()
}
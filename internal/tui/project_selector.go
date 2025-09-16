package tui

import (
	"fmt"
	"strings"

	"bqui/internal/bigquery"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

type ProjectSelectorModel struct {
	projects         []*bigquery.Project
	cursor           int
	filter           string
	filteredProjects []*bigquery.Project
}

func NewProjectSelectorModel() ProjectSelectorModel {
	return ProjectSelectorModel{
		projects:         make([]*bigquery.Project, 0),
		cursor:           0,
		filter:           "",
		filteredProjects: make([]*bigquery.Project, 0),
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
		m.updateFilteredProjects() // Initialize filtered projects
		return m, nil
	}
	return m, nil
}

func (m ProjectSelectorModel) handleKeypress(msg tea.KeyMsg) (ProjectSelectorModel, tea.Cmd) {
	switch {
	case key.Matches(msg, DefaultKeyMap().Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, DefaultKeyMap().Down):
		if m.cursor < len(m.filteredProjects)-1 {
			m.cursor++
		}

	case key.Matches(msg, DefaultKeyMap().Enter):
		if len(m.filteredProjects) > 0 && m.cursor < len(m.filteredProjects) {
			selectedProject := m.filteredProjects[m.cursor]
			return m, func() tea.Msg {
				return ProjectSelectedMsg{Project: selectedProject}
			}
		}
		return m, nil

	case msg.Type == tea.KeyBackspace:
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.updateFilteredProjects()
			m.cursor = 0 // Reset cursor when filter changes
		}

	case msg.Type == tea.KeyRunes:
		// Add typed characters to filter
		m.filter += string(msg.Runes)
		m.updateFilteredProjects()
		m.cursor = 0 // Reset cursor when filter changes
	}

	return m, nil
}

// updateFilteredProjects applies fuzzy search to filter projects
func (m *ProjectSelectorModel) updateFilteredProjects() {
	if m.filter == "" {
		m.filteredProjects = m.projects
		return
	}

	// Create search targets for fuzzy matching
	var targets []string
	for _, project := range m.projects {
		// Search in both project ID and name
		searchTarget := project.ID
		if project.Name != project.ID {
			searchTarget += " " + project.Name
		}
		targets = append(targets, searchTarget)
	}

	// Perform fuzzy search
	matches := fuzzy.Find(m.filter, targets)

	// Convert matches back to projects
	m.filteredProjects = make([]*bigquery.Project, 0, len(matches))
	for _, match := range matches {
		m.filteredProjects = append(m.filteredProjects, m.projects[match.Index])
	}
}

// Legacy method for compatibility - now just returns cached filtered projects
func (m ProjectSelectorModel) getFilteredProjects() []*bigquery.Project {
	return m.filteredProjects
}

func (m ProjectSelectorModel) View() string {
	var content strings.Builder

	content.WriteString(HeaderStyle.Render("🚀 Select Project") + "\n\n")

	// Show search input with cursor
	searchPrompt := "> " + m.filter + "█"
	content.WriteString(SelectedItemStyle.Render(searchPrompt) + "\n\n")

	if len(m.filteredProjects) == 0 {
		if m.filter != "" {
			content.WriteString(SubtleItemStyle.Render("No matching projects found for: " + m.filter))
		} else {
			content.WriteString(SubtleItemStyle.Render("No projects available"))
		}
		return content.String()
	}

	// Show up to 10 matches to keep it manageable like fzf
	maxVisible := 10
	if len(m.filteredProjects) < maxVisible {
		maxVisible = len(m.filteredProjects)
	}

	for i := 0; i < maxVisible; i++ {
		project := m.filteredProjects[i]
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

	// Show more indicator if there are additional matches
	if len(m.filteredProjects) > maxVisible {
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("  ... and %d more matches", len(m.filteredProjects)-maxVisible)) + "\n")
	}

	content.WriteString("\n" + SubtleItemStyle.Render(fmt.Sprintf("Matches: %d/%d", len(m.filteredProjects), len(m.projects))))
	content.WriteString("\n" + HelpStyle.Render("Type to search • ↑/↓ to navigate • Enter to select • Esc to cancel"))

	return content.String()
}

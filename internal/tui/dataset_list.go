package tui

import (
	"fmt"
	"strings"

	"bqui/internal/bigquery"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type DatasetListModel struct {
	datasets        []*bigquery.Dataset
	tables          []*bigquery.Table
	selectedDataset *bigquery.Dataset
	selectedTable   *bigquery.Table
	cursor          int
	filter          string
	showingTables   bool
	viewOffset      int
	tableSelected   bool
}

func NewDatasetListModel() DatasetListModel {
	return DatasetListModel{
		datasets:        make([]*bigquery.Dataset, 0),
		tables:          make([]*bigquery.Table, 0),
		selectedDataset: nil,
		selectedTable:   nil,
		cursor:          0,
		filter:          "",
		showingTables:   false,
		viewOffset:      0,
		tableSelected:   false,
	}
}

func (m DatasetListModel) Init() tea.Cmd {
	return nil
}

func (m DatasetListModel) Update(msg tea.Msg) (DatasetListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeypress(msg)
	}
	return m, nil
}

func (m DatasetListModel) handleKeypress(msg tea.KeyMsg) (DatasetListModel, tea.Cmd) {
	filteredItems := m.getFilteredItems()
	if len(filteredItems) == 0 {
		return m, nil
	}

	switch {
	case key.Matches(msg, DefaultKeyMap().Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, DefaultKeyMap().Down):
		if m.cursor < len(filteredItems)-1 {
			m.cursor++
		}

	case key.Matches(msg, DefaultKeyMap().VimTop), key.Matches(msg, DefaultKeyMap().Top):
		m.cursor = 0
		m.viewOffset = 0

	case key.Matches(msg, DefaultKeyMap().VimBottom), key.Matches(msg, DefaultKeyMap().Bottom):
		m.cursor = len(filteredItems) - 1
		// Scroll to show the bottom item
		maxVisible := 20
		if len(filteredItems) > maxVisible {
			m.viewOffset = len(filteredItems) - maxVisible
		} else {
			m.viewOffset = 0
		}

	case key.Matches(msg, DefaultKeyMap().PageUp):
		m.cursor -= 10
		if m.cursor < 0 {
			m.cursor = 0
		}

	case key.Matches(msg, DefaultKeyMap().PageDown):
		m.cursor += 10
		if m.cursor >= len(filteredItems) {
			m.cursor = len(filteredItems) - 1
		}

	case key.Matches(msg, DefaultKeyMap().Enter):
		if !m.showingTables {
			if m.cursor < len(m.getFilteredDatasets()) {
				m.selectedDataset = m.getFilteredDatasets()[m.cursor]
				m.showingTables = true
				m.cursor = 0
				m.selectedTable = nil // Reset table selection
				return m, nil
			}
		} else {
			if m.cursor < len(m.getFilteredTables()) {
				m.selectedTable = m.getFilteredTables()[m.cursor]
				m.tableSelected = true // Set flag to indicate explicit selection
				return m, nil
			}
		}

	case key.Matches(msg, DefaultKeyMap().Left):
		if m.showingTables {
			m.showingTables = false
			m.selectedTable = nil
			m.cursor = 0
		}

	case key.Matches(msg, DefaultKeyMap().Right):
		if !m.showingTables && m.selectedDataset != nil {
			m.showingTables = true
			m.cursor = 0
		}
	}

	m.updateSelection()
	m.ensureCursorVisible(len(filteredItems))
	return m, nil
}

func (m *DatasetListModel) updateSelection() {
	if !m.showingTables {
		datasets := m.getFilteredDatasets()
		if m.cursor < len(datasets) {
			m.selectedDataset = datasets[m.cursor]
		}
	} else {
		tables := m.getFilteredTables()
		if m.cursor < len(tables) {
			m.selectedTable = tables[m.cursor]
		}
	}
}

func (m DatasetListModel) getFilteredItems() []string {
	if !m.showingTables {
		var items []string
		for _, dataset := range m.getFilteredDatasets() {
			items = append(items, dataset.ID)
		}
		return items
	} else {
		var items []string
		for _, table := range m.getFilteredTables() {
			items = append(items, table.ID)
		}
		return items
	}
}

func (m DatasetListModel) getFilteredDatasets() []*bigquery.Dataset {
	if m.filter == "" {
		return m.datasets
	}

	var filtered []*bigquery.Dataset
	for _, dataset := range m.datasets {
		if strings.Contains(strings.ToLower(dataset.ID), strings.ToLower(m.filter)) {
			filtered = append(filtered, dataset)
		}
	}
	return filtered
}

func (m DatasetListModel) getFilteredTables() []*bigquery.Table {
	if m.filter == "" {
		return m.tables
	}

	var filtered []*bigquery.Table
	for _, table := range m.tables {
		if strings.Contains(strings.ToLower(table.ID), strings.ToLower(m.filter)) {
			filtered = append(filtered, table)
		}
	}
	return filtered
}

func (m *DatasetListModel) ensureCursorVisible(totalItems int) {
	maxVisible := 20
	
	// If cursor is above the visible area, scroll up
	if m.cursor < m.viewOffset {
		m.viewOffset = m.cursor
	}
	
	// If cursor is below the visible area, scroll down
	if m.cursor >= m.viewOffset+maxVisible {
		m.viewOffset = m.cursor - maxVisible + 1
	}
	
	// Make sure viewOffset doesn't go negative
	if m.viewOffset < 0 {
		m.viewOffset = 0
	}
	
	// Make sure we don't scroll past the end
	if totalItems > maxVisible && m.viewOffset > totalItems-maxVisible {
		m.viewOffset = totalItems - maxVisible
	}
}

func (m DatasetListModel) View() string {
	var content strings.Builder

	title := "📊 Datasets"
	if m.showingTables {
		title = fmt.Sprintf("📋 Tables in %s", m.selectedDataset.ID)
	}

	content.WriteString(HeaderStyle.Render(title) + "\n\n")

	if m.filter != "" {
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("Filter: %s", m.filter)) + "\n\n")
	}

	filteredItems := m.getFilteredItems()
	if len(filteredItems) == 0 {
		if len(m.datasets) == 0 && !m.showingTables {
			content.WriteString(SubtleItemStyle.Render("No datasets found"))
		} else if len(m.tables) == 0 && m.showingTables {
			content.WriteString(SubtleItemStyle.Render("No tables found"))
		} else {
			content.WriteString(SubtleItemStyle.Render("No items match filter"))
		}
		return content.String()
	}

	visibleStart := m.viewOffset
	visibleEnd := visibleStart + 20

	if visibleEnd > len(filteredItems) {
		visibleEnd = len(filteredItems)
	}

	for i := visibleStart; i < visibleEnd; i++ {
		item := filteredItems[i]
		style := ItemStyle

		if i == m.cursor {
			style = SelectedItemStyle
		}

		prefix := "  "
		if m.showingTables {
			prefix = "  🗂  "
		} else {
			prefix = "  📁 "
		}

		content.WriteString(style.Render(prefix + item) + "\n")
	}

	if len(filteredItems) > visibleEnd {
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("... and %d more", len(filteredItems)-visibleEnd)) + "\n")
	}

	info := ""
	if m.showingTables {
		info = fmt.Sprintf("Tables: %d", len(filteredItems))
	} else {
		info = fmt.Sprintf("Datasets: %d", len(filteredItems))
	}

	content.WriteString("\n" + SubtleItemStyle.Render(info))

	return content.String()
}
package tui

import (
	"fmt"
	"strings"

	"bqui/internal/bigquery"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TabType int

const (
	SchemaTab TabType = iota
	PreviewTab
	QueryTab
)

type TableDetailModel struct {
	schema           *bigquery.TableSchema
	preview          *bigquery.TablePreview
	queryResult      *bigquery.QueryResult
	activeTab        TabType
	queryInput       textarea.Model
	scrollOffset     int
	horizontalOffset int
	currentTableName string
	schemaFilter     string
	showSchemaFilter bool
	previewRowCursor int
	previewColCursor int
	schemaRowCursor  int
}

func NewTableDetailModel() TableDetailModel {
	queryInput := textarea.New()
	queryInput.Placeholder = "Enter your SQL query here..."
	queryInput.Focus()

	return TableDetailModel{
		schema:           nil,
		preview:          nil,
		queryResult:      nil,
		activeTab:        SchemaTab,
		queryInput:       queryInput,
		scrollOffset:     0,
		horizontalOffset: 0,
		schemaFilter:     "",
		showSchemaFilter: false,
		previewRowCursor: 0,
		previewColCursor: 0,
		schemaRowCursor:  0,
	}
}

func (m TableDetailModel) Init() tea.Cmd {
	return nil
}

func (m TableDetailModel) Update(msg tea.Msg) (TableDetailModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle schema filter input
		if m.showSchemaFilter {
			switch msg.String() {
			case "enter":
				m.showSchemaFilter = false
				m.schemaRowCursor = 0 // Reset cursor when filter is applied
				return m, nil
			case "esc":
				m.showSchemaFilter = false
				m.schemaFilter = ""
				m.schemaRowCursor = 0 // Reset cursor when filter is cleared
				return m, nil
			case "backspace":
				if len(m.schemaFilter) > 0 {
					m.schemaFilter = m.schemaFilter[:len(m.schemaFilter)-1]
				}
				m.schemaRowCursor = 0 // Reset cursor when filter changes
				return m, nil
			default:
				if len(msg.String()) == 1 {
					m.schemaFilter += msg.String()
					m.schemaRowCursor = 0 // Reset cursor when filter changes
				}
				return m, nil
			}
		}

		if m.activeTab == QueryTab && m.queryInput.Focused() {
			switch {
			case key.Matches(msg, DefaultKeyMap().Escape):
				m.queryInput.Blur()
				return m, nil
			case key.Matches(msg, DefaultKeyMap().Tab):
				// Tab should cycle tabs, not be consumed by textarea
				m.activeTab = TabType((int(m.activeTab) + 1) % 3)
				m.scrollOffset = 0
				m.queryInput.Blur() // Blur the input when switching tabs
				return m, nil
			case key.Matches(msg, DefaultKeyMap().ShiftTab):
				// Shift+Tab cycles backward
				m.activeTab = TabType((int(m.activeTab) + 2) % 3) // +2 is same as -1 in mod 3
				m.scrollOffset = 0
				m.queryInput.Blur() // Blur the input when switching tabs
				return m, nil
			default:
				m.queryInput, cmd = m.queryInput.Update(msg)
				return m, cmd
			}
		}

		return m.handleKeypress(msg)

	case QueryResultMsg:
		m.queryResult = msg.Result
		return m, nil
	}

	return m, cmd
}

func (m TableDetailModel) handleKeypress(msg tea.KeyMsg) (TableDetailModel, tea.Cmd) {
	switch {
	case key.Matches(msg, DefaultKeyMap().Tab):
		m.activeTab = TabType((int(m.activeTab) + 1) % 3)
		m.scrollOffset = 0
		m.previewRowCursor = 0
		m.previewColCursor = 0
		m.schemaRowCursor = 0

	case key.Matches(msg, DefaultKeyMap().ShiftTab):
		m.activeTab = TabType((int(m.activeTab) + 2) % 3) // +2 is same as -1 in mod 3
		m.scrollOffset = 0
		m.previewRowCursor = 0
		m.previewColCursor = 0
		m.schemaRowCursor = 0

	case key.Matches(msg, DefaultKeyMap().Up):
		if m.activeTab == PreviewTab && m.preview != nil {
			if m.previewRowCursor > 0 {
				m.previewRowCursor--
			}
		} else if m.activeTab == SchemaTab && m.schema != nil {
			if m.schemaRowCursor > 0 {
				m.schemaRowCursor--
			}
		} else {
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		}

	case key.Matches(msg, DefaultKeyMap().Down):
		if m.activeTab == PreviewTab && m.preview != nil {
			if m.previewRowCursor < len(m.preview.Rows)-1 {
				m.previewRowCursor++
			}
		} else if m.activeTab == SchemaTab && m.schema != nil {
			filteredFields := m.getFilteredSchemaFields()
			if m.schemaRowCursor < len(filteredFields)-1 {
				m.schemaRowCursor++
			}
		} else {
			m.scrollOffset++
		}

	case key.Matches(msg, DefaultKeyMap().VimTop), key.Matches(msg, DefaultKeyMap().Top):
		if m.activeTab == PreviewTab && m.preview != nil {
			m.previewRowCursor = 0
			m.previewColCursor = 0
		} else if m.activeTab == SchemaTab && m.schema != nil {
			m.schemaRowCursor = 0
		} else {
			m.scrollOffset = 0
		}

	case key.Matches(msg, DefaultKeyMap().VimBottom), key.Matches(msg, DefaultKeyMap().Bottom):
		if m.activeTab == SchemaTab && m.schema != nil {
			filteredFields := m.getFilteredSchemaFields()
			if len(filteredFields) > 0 {
				m.schemaRowCursor = len(filteredFields) - 1
			}
		} else if m.activeTab == PreviewTab && m.preview != nil {
			if len(m.preview.Rows) > 0 {
				m.previewRowCursor = len(m.preview.Rows) - 1
			} else {
				m.previewRowCursor = 0
			}
			if len(m.preview.Headers) > 0 {
				m.previewColCursor = len(m.preview.Headers) - 1
			} else {
				m.previewColCursor = 0
			}
		} else if m.activeTab == QueryTab && m.queryResult != nil {
			maxVisible := 10
			if len(m.queryResult.Rows) > maxVisible {
				m.scrollOffset = len(m.queryResult.Rows) - maxVisible
			} else {
				m.scrollOffset = 0
			}
		}

	case key.Matches(msg, DefaultKeyMap().PageUp):
		if m.activeTab == PreviewTab && m.preview != nil {
			m.previewRowCursor -= 10
			if m.previewRowCursor < 0 {
				m.previewRowCursor = 0
			}
		} else if m.activeTab == SchemaTab && m.schema != nil {
			m.schemaRowCursor -= 10
			if m.schemaRowCursor < 0 {
				m.schemaRowCursor = 0
			}
		} else {
			m.scrollOffset -= 10
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		}

	case key.Matches(msg, DefaultKeyMap().PageDown):
		if m.activeTab == PreviewTab && m.preview != nil {
			m.previewRowCursor += 10
			if m.previewRowCursor >= len(m.preview.Rows) {
				m.previewRowCursor = len(m.preview.Rows) - 1
			}
		} else if m.activeTab == SchemaTab && m.schema != nil {
			filteredFields := m.getFilteredSchemaFields()
			m.schemaRowCursor += 10
			if m.schemaRowCursor >= len(filteredFields) {
				m.schemaRowCursor = len(filteredFields) - 1
			}
		} else {
			m.scrollOffset += 10
		}

	case key.Matches(msg, DefaultKeyMap().Left):
		if m.activeTab == PreviewTab && m.preview != nil {
			if m.previewColCursor > 0 {
				m.previewColCursor--
			}
		} else {
			if m.horizontalOffset > 0 {
				m.horizontalOffset--
			}
		}

	case key.Matches(msg, DefaultKeyMap().Right):
		if m.activeTab == PreviewTab && m.preview != nil {
			if m.previewColCursor < len(m.preview.Headers)-1 {
				m.previewColCursor++
			}
		} else {
			m.horizontalOffset++
		}

	case key.Matches(msg, DefaultKeyMap().Search):
		if m.activeTab == SchemaTab {
			m.showSchemaFilter = true
			return m, nil
		}

	case key.Matches(msg, DefaultKeyMap().Escape):
		if m.showSchemaFilter {
			m.showSchemaFilter = false
			m.schemaFilter = ""
			return m, nil
		}

	case key.Matches(msg, DefaultKeyMap().Enter):
		if m.activeTab == QueryTab && !m.queryInput.Focused() {
			m.queryInput.Focus()
			return m, nil
		}
	}

	// Ensure cursors are visible
	if m.activeTab == PreviewTab && m.preview != nil {
		m.ensurePreviewCursorVisible()
		m.ensurePreviewColumnVisible()
	} else if m.activeTab == SchemaTab && m.schema != nil {
		m.ensureSchemaCursorVisible()
	}

	return m, nil
}

func (m *TableDetailModel) ensurePreviewCursorVisible() {
	maxVisible := 15

	// If cursor is above the visible area, scroll up
	if m.previewRowCursor < m.scrollOffset {
		m.scrollOffset = m.previewRowCursor
	}

	// If cursor is below the visible area, scroll down
	if m.previewRowCursor >= m.scrollOffset+maxVisible {
		m.scrollOffset = m.previewRowCursor - maxVisible + 1
	}

	// Make sure scrollOffset doesn't go negative
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	// Make sure we don't scroll past the end
	if len(m.preview.Rows) > maxVisible && m.scrollOffset > len(m.preview.Rows)-maxVisible {
		m.scrollOffset = len(m.preview.Rows) - maxVisible
	}
}

func (m *TableDetailModel) ensurePreviewColumnVisible() {
	if len(m.preview.Headers) == 0 {
		return
	}

	// Calculate column positions for horizontal scrolling
	colWidths := make([]int, len(m.preview.Headers))
	minColWidth := 8
	maxColWidth := 30

	// Calculate width based on header and data
	for i, header := range m.preview.Headers {
		width := len(header)
		for _, row := range m.preview.Rows {
			if i < len(row) {
				cellStr := fmt.Sprintf("%v", row[i])
				if len(cellStr) > width {
					width = len(cellStr)
				}
			}
		}
		if width < minColWidth {
			width = minColWidth
		}
		if width > maxColWidth {
			width = maxColWidth
		}
		colWidths[i] = width
	}

	// Calculate start position of selected column
	selectedColStart := 0
	for i := 0; i < m.previewColCursor; i++ {
		selectedColStart += colWidths[i] + 1 // +1 for space
	}

	// Calculate end position of selected column
	selectedColEnd := selectedColStart + colWidths[m.previewColCursor]

	// Adjust horizontal offset to keep selected column visible
	maxDisplayWidth := 80 // Approximate display width for cells

	// If selected column starts before visible area, scroll left
	if selectedColStart < m.horizontalOffset {
		m.horizontalOffset = selectedColStart
	}

	// If selected column ends after visible area, scroll right
	if selectedColEnd > m.horizontalOffset+maxDisplayWidth {
		m.horizontalOffset = selectedColEnd - maxDisplayWidth
		if m.horizontalOffset < 0 {
			m.horizontalOffset = 0
		}
	}
}

func (m *TableDetailModel) ensureSchemaCursorVisible() {
	filteredFields := m.getFilteredSchemaFields()
	maxVisible := 15

	// If cursor is above the visible area, scroll up
	if m.schemaRowCursor < m.scrollOffset {
		m.scrollOffset = m.schemaRowCursor
	}

	// If cursor is below the visible area, scroll down
	if m.schemaRowCursor >= m.scrollOffset+maxVisible {
		m.scrollOffset = m.schemaRowCursor - maxVisible + 1
	}

	// Make sure scrollOffset doesn't go negative
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	// Make sure we don't scroll past the end
	if len(filteredFields) > maxVisible && m.scrollOffset > len(filteredFields)-maxVisible {
		m.scrollOffset = len(filteredFields) - maxVisible
	}
}

func (m TableDetailModel) View() string {
	var content strings.Builder

	tabs := m.renderTabs()
	content.WriteString(tabs + "\n\n")

	switch m.activeTab {
	case SchemaTab:
		content.WriteString(m.renderSchemaTab())
	case PreviewTab:
		content.WriteString(m.renderPreviewTab())
	case QueryTab:
		content.WriteString(m.renderQueryTab())
	}

	return content.String()
}

func (m TableDetailModel) renderTabs() string {
	var tabs []string

	schemaStyle := TabInactiveStyle
	previewStyle := TabInactiveStyle
	queryStyle := TabInactiveStyle

	switch m.activeTab {
	case SchemaTab:
		schemaStyle = TabActiveStyle
	case PreviewTab:
		previewStyle = TabActiveStyle
	case QueryTab:
		queryStyle = TabActiveStyle
	}

	tabs = append(tabs, schemaStyle.Render("Schema"))
	tabs = append(tabs, previewStyle.Render("Preview"))
	tabs = append(tabs, queryStyle.Render("Query"))

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m TableDetailModel) renderSchemaTab() string {
	if m.schema == nil {
		return SubtleItemStyle.Render("No schema loaded. Select a table to view its schema.")
	}

	var content strings.Builder
	content.WriteString(HeaderStyle.Render("🏗  Table Schema") + "\n")

	if m.currentTableName != "" {
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("Table: %s", m.currentTableName)) + "\n")
	}

	// Show schema filter if active
	if m.showSchemaFilter {
		content.WriteString(SearchBoxStyle.Render(fmt.Sprintf("Search columns: %s", m.schemaFilter)) + "\n")
	} else if m.schemaFilter != "" {
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("Filter: %s (press / to edit, esc to clear)", m.schemaFilter)) + "\n")
	}
	content.WriteString("\n")

	// Render tabular header
	headerLine := fmt.Sprintf("%-25s %-15s %-10s %s", "Field Name", "Type", "Mode", "Description")
	content.WriteString(HeaderStyle.Render(headerLine) + "\n")
	content.WriteString(strings.Repeat("─", 80) + "\n")

	visible := 0
	maxVisible := 15
	filteredFields := m.getFilteredSchemaFields()

	for i, field := range filteredFields {
		if visible-m.scrollOffset < 0 {
			visible++
			continue
		}
		if visible-m.scrollOffset >= maxVisible {
			break
		}

		fieldStr := m.renderSchemaFieldTabular(field, 0)

		// Highlight selected row
		if i == m.schemaRowCursor {
			content.WriteString(SelectedItemStyle.Render(fieldStr) + "\n")
		} else {
			content.WriteString(fieldStr + "\n")
		}
		visible++
	}

	if len(filteredFields) > m.scrollOffset+maxVisible {
		remaining := len(filteredFields) - (m.scrollOffset + maxVisible)
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("\n... and %d more fields", remaining)) + "\n")
	}

	// Add help text
	if !m.showSchemaFilter {
		content.WriteString("\n" + HelpStyle.Render("Press / to search columns, ←→ or h/l to scroll horizontally"))
	}

	return content.String()
}

func (m TableDetailModel) renderPreviewTab() string {
	if m.preview == nil {
		return SubtleItemStyle.Render("No preview loaded. Select a table to view sample data.")
	}

	var content strings.Builder
	content.WriteString(HeaderStyle.Render("👀 Table Preview") + "\n")

	if m.currentTableName != "" {
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("Table: %s", m.currentTableName)) + "\n\n")
	} else {
		content.WriteString("\n")
	}

	if len(m.preview.Headers) == 0 {
		content.WriteString(SubtleItemStyle.Render("No data available"))
		return content.String()
	}

	// Calculate column widths for proper alignment
	colWidths := make([]int, len(m.preview.Headers))
	minColWidth := 8
	maxColWidth := 30

	// Calculate width based on header and data
	for i, header := range m.preview.Headers {
		width := len(header)
		for _, row := range m.preview.Rows {
			if i < len(row) {
				cellStr := fmt.Sprintf("%v", row[i])
				if len(cellStr) > width {
					width = len(cellStr)
				}
			}
		}
		if width < minColWidth {
			width = minColWidth
		}
		if width > maxColWidth {
			width = maxColWidth
		}
		colWidths[i] = width
	}

	// Render header with proper alignment and horizontal scrolling
	headerRow := ""
	currentPos := 0
	for i, header := range m.preview.Headers {
		headerText := truncate(header, colWidths[i])
		cellFormatted := HeaderStyle.Render(fmt.Sprintf("%-*s", colWidths[i], headerText)) + " "

		// Check if this column should be visible based on horizontal offset
		cellStart := currentPos
		cellEnd := currentPos + len(cellFormatted)

		if cellEnd > m.horizontalOffset && cellStart < m.horizontalOffset+80 {
			// Column is visible, add to row
			if cellStart < m.horizontalOffset {
				// Partially visible from left
				visibleStart := m.horizontalOffset - cellStart
				if visibleStart < len(cellFormatted) {
					headerRow += cellFormatted[visibleStart:]
				}
			} else {
				headerRow += cellFormatted
			}
		}
		currentPos = cellEnd
	}
	content.WriteString(headerRow + "\n")

	// Add separator
	separatorWidth := len(headerRow)
	if separatorWidth > 80 {
		separatorWidth = 80
	}
	content.WriteString(strings.Repeat("─", separatorWidth) + "\n")

	// Calculate visible rows based on scroll and display
	maxVisible := 15
	startRow := m.scrollOffset
	endRow := startRow + maxVisible
	if endRow > len(m.preview.Rows) {
		endRow = len(m.preview.Rows)
	}

	// Render visible rows
	for rowIdx := startRow; rowIdx < endRow; rowIdx++ {
		row := m.preview.Rows[rowIdx]
		currentPos := 0

		for i, cell := range row {
			if i >= len(m.preview.Headers) {
				break
			}
			cellStr := fmt.Sprintf("%v", cell)
			cellText := truncate(cellStr, colWidths[i])
			cellFormatted := fmt.Sprintf("%-*s", colWidths[i], cellText)

			// Apply styling and check visibility
			var styledCell string
			if rowIdx == m.previewRowCursor && i == m.previewColCursor {
				styledCell = SelectedItemStyle.Render(cellFormatted) + " "
			} else {
				styledCell = TableCellStyle.Render(cellFormatted) + " "
			}

			// Check if this column should be visible based on horizontal offset
			cellStart := currentPos
			cellEnd := currentPos + len(styledCell)

			if cellEnd > m.horizontalOffset && cellStart < m.horizontalOffset+80 {
				// Column is visible, add to row
				if cellStart < m.horizontalOffset {
					// Partially visible from left
					visibleStart := m.horizontalOffset - cellStart
					if visibleStart < len(styledCell) {
						content.WriteString(styledCell[visibleStart:])
					}
				} else {
					content.WriteString(styledCell)
				}
			}
			currentPos = cellEnd
		}
		content.WriteString("\n")
	}

	// Show remaining rows indicator
	if endRow < len(m.preview.Rows) {
		remaining := len(m.preview.Rows) - endRow
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("\n... and %d more rows", remaining)) + "\n")
	}

	// Show info and help
	selectedColumnName := ""
	if len(m.preview.Headers) > m.previewColCursor {
		selectedColumnName = m.preview.Headers[m.previewColCursor]
	}
	info := fmt.Sprintf("Showing rows %d-%d of %d | Cell [%d,%d] (%s) selected",
		startRow+1, endRow, len(m.preview.Rows), m.previewRowCursor+1, m.previewColCursor+1, selectedColumnName)
	content.WriteString("\n" + SubtleItemStyle.Render(info))
	content.WriteString("\n" + HelpStyle.Render("Press y to copy selected cell, arrow keys/hjkl to navigate cells"))

	return content.String()
}

func (m TableDetailModel) renderQueryTab() string {
	var content strings.Builder
	content.WriteString(HeaderStyle.Render("🔍 Query") + "\n")

	if m.currentTableName != "" {
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("Table: %s", m.currentTableName)) + "\n\n")
	} else {
		content.WriteString("\n")
	}

	content.WriteString("SQL Query:\n")
	content.WriteString(m.queryInput.View() + "\n")

	if !m.queryInput.Focused() {
		content.WriteString(HelpStyle.Render("Press Enter to edit query, Esc to exit edit mode") + "\n")
	}

	if m.queryResult != nil {
		content.WriteString("\n" + HeaderStyle.Render("Query Results:") + "\n\n")

		if len(m.queryResult.Columns) > 0 {
			headerRow := ""
			for _, col := range m.queryResult.Columns {
				headerRow += HeaderStyle.Render(fmt.Sprintf("%-20s", truncate(col, 20))) + " "
			}
			content.WriteString(headerRow + "\n")
			content.WriteString(strings.Repeat("-", len(headerRow)) + "\n")

			visible := 0
			maxVisible := 10

			for _, row := range m.queryResult.Rows {
				if visible-m.scrollOffset < 0 {
					visible++
					continue
				}
				if visible-m.scrollOffset >= maxVisible {
					break
				}

				rowStr := ""
				for _, cell := range row {
					cellStr := fmt.Sprintf("%v", cell)
					rowStr += TableCellStyle.Render(fmt.Sprintf("%-20s", truncate(cellStr, 20))) + " "
				}
				content.WriteString(rowStr + "\n")
				visible++
			}

			if len(m.queryResult.Rows) > maxVisible {
				content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("... and %d more rows", len(m.queryResult.Rows)-maxVisible)) + "\n")
			}
		}
	}

	return content.String()
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

func (m TableDetailModel) getFilteredSchemaFields() []*bigquery.Column {
	if m.schema == nil {
		return nil
	}

	if m.schemaFilter == "" {
		return m.schema.Fields
	}

	var filtered []*bigquery.Column
	for _, field := range m.schema.Fields {
		if m.fieldMatchesFilter(field, strings.ToLower(m.schemaFilter)) {
			filtered = append(filtered, field)
		}
	}
	return filtered
}

func (m TableDetailModel) fieldMatchesFilter(field *bigquery.Column, filter string) bool {
	// Check field name
	if strings.Contains(strings.ToLower(field.Name), filter) {
		return true
	}

	// Check field description
	if strings.Contains(strings.ToLower(field.Description), filter) {
		return true
	}

	// Check field type
	if strings.Contains(strings.ToLower(string(field.Type)), filter) {
		return true
	}

	// Check nested fields recursively
	for _, subField := range field.Fields {
		if m.fieldMatchesFilter(subField, filter) {
			return true
		}
	}

	return false
}

func (m TableDetailModel) renderSchemaFieldTabular(field *bigquery.Column, indent int) string {
	indentStr := strings.Repeat("  ", indent)

	// Format mode
	mode := "NULLABLE"
	if field.Required {
		mode = "REQUIRED"
	} else if field.Repeated {
		mode = "REPEATED"
	}

	// Truncate description if too long
	desc := field.Description
	if len(desc) > 40 {
		desc = desc[:37] + "..."
	}

	// Apply horizontal offset for scrolling
	fullLine := fmt.Sprintf("%s%-25s %-15s %-10s %s",
		indentStr,
		truncate(field.Name, 25),
		DataTypeStyle.Render(truncate(string(field.Type), 15)),
		SubtleItemStyle.Render(mode),
		SubtleItemStyle.Render(desc))

	// Apply horizontal scrolling
	if m.horizontalOffset > 0 && len(fullLine) > m.horizontalOffset {
		fullLine = fullLine[m.horizontalOffset:]
	}

	result := fullLine

	// Render nested fields
	for _, subField := range field.Fields {
		result += "\n" + m.renderSchemaFieldTabular(subField, indent+1)
	}

	return result
}

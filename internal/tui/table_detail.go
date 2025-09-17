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
	ResultsTab
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
	currentProjectID string
	currentDatasetID string
	schemaFilter     string
	showSchemaFilter bool
	previewRowCursor int
	previewColCursor int
	schemaRowCursor  int
	height           int
	width            int
	// Visual selection mode
	visualMode     bool
	visualStartRow int
	visualEndRow   int
	// Preview filtering
	previewFilter     string
	showPreviewFilter bool
	// Dialog and Results
	showColumnDialog bool
	selectedColumn   *bigquery.Column
	dialogCursor     int
	queryResults     *bigquery.QueryResult
	executedQuery    string
	resultsRowCursor int
	resultsColCursor int
}

func NewTableDetailModel() TableDetailModel {
	queryInput := textarea.New()
	queryInput.Placeholder = "Enter your SQL query here..."
	queryInput.Focus()

	return TableDetailModel{
		schema:            nil,
		preview:           nil,
		queryResult:       nil,
		activeTab:         SchemaTab,
		queryInput:        queryInput,
		scrollOffset:      0,
		horizontalOffset:  0,
		currentTableName:  "",
		currentProjectID:  "",
		currentDatasetID:  "",
		schemaFilter:      "",
		showSchemaFilter:  false,
		previewRowCursor:  0,
		previewColCursor:  0,
		schemaRowCursor:   0,
		height:            20,
		visualMode:        false,
		visualStartRow:    0,
		visualEndRow:      0,
		previewFilter:     "",
		showPreviewFilter: false,
		showColumnDialog:  false,
		selectedColumn:    nil,
		dialogCursor:      0,
		queryResults:      nil,
		executedQuery:     "",
		resultsRowCursor:  0,
		resultsColCursor:  0,
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

		// Handle preview filter input
		if m.showPreviewFilter {
			switch msg.String() {
			case "enter":
				m.showPreviewFilter = false
				m.previewRowCursor = 0 // Reset cursor when filter is applied
				return m, nil
			case "esc":
				m.showPreviewFilter = false
				m.previewFilter = ""
				m.previewRowCursor = 0 // Reset cursor when filter is cleared
				return m, nil
			case "backspace":
				if len(m.previewFilter) > 0 {
					m.previewFilter = m.previewFilter[:len(m.previewFilter)-1]
				}
				m.previewRowCursor = 0 // Reset cursor when filter changes
				return m, nil
			default:
				if len(msg.String()) == 1 {
					m.previewFilter += msg.String()
					m.previewRowCursor = 0 // Reset cursor when filter changes
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
	// Handle ESC key with proper hierarchy
	if key.Matches(msg, DefaultKeyMap().Escape) {
		return m.handleEscapeKey()
	}

	// Handle visual mode keys
	if msg.String() == "V" || msg.String() == "shift+v" {
		if (m.activeTab == PreviewTab && m.preview != nil) || (m.activeTab == ResultsTab && m.queryResults != nil) {
			m.visualMode = !m.visualMode
			if m.visualMode {
				switch m.activeTab {
				case PreviewTab:
					m.visualStartRow = m.previewRowCursor
					m.visualEndRow = m.previewRowCursor
				case ResultsTab:
					m.visualStartRow = m.resultsRowCursor
					m.visualEndRow = m.resultsRowCursor
				}
			} else {
				m.visualStartRow = 0
				m.visualEndRow = 0
			}
		}
		return m, nil
	}

	// Handle search trigger
	if msg.String() == "/" {
		switch m.activeTab {
		case SchemaTab:
			m.showSchemaFilter = true
			return m, nil
		case PreviewTab:
			m.showPreviewFilter = true
			return m, nil
		}
	}

	// Handle horizontal navigation shortcuts
	if msg.String() == "0" {
		switch m.activeTab {
		case PreviewTab:
			m.previewColCursor = 0
			// Force horizontal scroll to show the first column
			m.forceScrollToColumn(0)
		case SchemaTab:
			m.horizontalOffset = 0
		}
		return m, nil
	}

	if msg.String() == "$" {
		if m.activeTab == PreviewTab && m.preview != nil {
			if len(m.preview.Headers) > 0 {
				m.previewColCursor = len(m.preview.Headers) - 1
				// Force horizontal scroll to show the last column immediately
				m.forceScrollToColumn(m.previewColCursor)
			}
		} else if m.activeTab == SchemaTab {
			// Move to end of schema view
			m.horizontalOffset = 1000 // Large value to scroll to end
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, DefaultKeyMap().Tab):
		m.activeTab = TabType((int(m.activeTab) + 1) % 4)
		m.scrollOffset = 0
		m.previewRowCursor = 0
		m.previewColCursor = 0
		m.schemaRowCursor = 0
		m.visualMode = false // Exit visual mode when changing tabs

	case key.Matches(msg, DefaultKeyMap().ShiftTab):
		m.activeTab = TabType((int(m.activeTab) + 3) % 4) // +3 is same as -1 in mod 4
		m.scrollOffset = 0
		m.previewRowCursor = 0
		m.previewColCursor = 0
		m.schemaRowCursor = 0
		m.visualMode = false // Exit visual mode when changing tabs

	case key.Matches(msg, DefaultKeyMap().Up):
		if m.showColumnDialog {
			if m.dialogCursor > 0 {
				m.dialogCursor--
			}
		} else if m.activeTab == PreviewTab && m.preview != nil {
			if m.previewRowCursor > 0 {
				m.previewRowCursor--
				// Update visual selection end if in visual mode
				if m.visualMode {
					m.visualEndRow = m.previewRowCursor
				}
			}
		} else if m.activeTab == SchemaTab && m.schema != nil {
			if m.schemaRowCursor > 0 {
				m.schemaRowCursor--
			}
		} else if m.activeTab == ResultsTab && m.queryResults != nil {
			if m.resultsRowCursor > 0 {
				m.resultsRowCursor--
				// Update visual selection end if in visual mode
				if m.visualMode {
					m.visualEndRow = m.resultsRowCursor
				}
			}
		} else {
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		}

	case key.Matches(msg, DefaultKeyMap().Down):
		if m.showColumnDialog {
			maxOptions := m.getDialogOptionCount()
			if m.dialogCursor < maxOptions-1 {
				m.dialogCursor++
			}
		} else if m.activeTab == PreviewTab && m.preview != nil {
			filteredRows := m.getFilteredPreviewRows()
			if m.previewRowCursor < len(filteredRows)-1 {
				m.previewRowCursor++
				// Update visual selection end if in visual mode
				if m.visualMode {
					m.visualEndRow = m.previewRowCursor
				}
			}
		} else if m.activeTab == SchemaTab && m.schema != nil {
			filteredFields := m.getFilteredSchemaFields()
			if m.schemaRowCursor < len(filteredFields)-1 {
				m.schemaRowCursor++
			}
		} else if m.activeTab == ResultsTab && m.queryResults != nil {
			if m.resultsRowCursor < len(m.queryResults.Rows)-1 {
				m.resultsRowCursor++
				// Update visual selection end if in visual mode
				if m.visualMode {
					m.visualEndRow = m.resultsRowCursor
				}
			}
		} else {
			m.scrollOffset++
		}

	case key.Matches(msg, DefaultKeyMap().VimTop), key.Matches(msg, DefaultKeyMap().Top):
		if m.activeTab == PreviewTab && m.preview != nil {
			m.previewRowCursor = 0
			// In visual mode, extend selection to top; otherwise reset column cursor
			if m.visualMode {
				m.visualEndRow = 0
			} else {
				m.previewColCursor = 0
			}
		} else if m.activeTab == ResultsTab && m.queryResults != nil {
			m.resultsRowCursor = 0
			// In visual mode, extend selection to top; otherwise reset column cursor
			if m.visualMode {
				m.visualEndRow = 0
			} else {
				m.resultsColCursor = 0
			}
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
			filteredRows := m.getFilteredPreviewRows()
			if len(filteredRows) > 0 {
				m.previewRowCursor = len(filteredRows) - 1
				// In visual mode, extend selection to bottom; otherwise move column cursor too
				if m.visualMode {
					m.visualEndRow = len(filteredRows) - 1
				} else {
					if len(m.preview.Headers) > 0 {
						m.previewColCursor = len(m.preview.Headers) - 1
					} else {
						m.previewColCursor = 0
					}
				}
			} else {
				m.previewRowCursor = 0
			}
		} else if m.activeTab == QueryTab && m.queryResult != nil {
			maxVisible := 10
			if len(m.queryResult.Rows) > maxVisible {
				m.scrollOffset = len(m.queryResult.Rows) - maxVisible
			} else {
				m.scrollOffset = 0
			}
		} else if m.activeTab == ResultsTab && m.queryResults != nil {
			if len(m.queryResults.Rows) > 0 {
				m.resultsRowCursor = len(m.queryResults.Rows) - 1
				// In visual mode, extend selection to bottom; otherwise move column cursor too
				if m.visualMode {
					m.visualEndRow = len(m.queryResults.Rows) - 1
				} else {
					if len(m.queryResults.Columns) > 0 {
						m.resultsColCursor = len(m.queryResults.Columns) - 1
					} else {
						m.resultsColCursor = 0
					}
				}
			} else {
				m.resultsRowCursor = 0
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
				// The horizontal offset will be automatically adjusted in the view rendering
			}
		} else if m.activeTab == ResultsTab && m.queryResults != nil {
			if m.resultsColCursor > 0 {
				m.resultsColCursor--
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
				// The horizontal offset will be automatically adjusted in the view rendering
			}
		} else if m.activeTab == ResultsTab && m.queryResults != nil {
			if m.resultsColCursor < len(m.queryResults.Columns)-1 {
				m.resultsColCursor++
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
		if m.showColumnDialog {
			// Execute the selected query option
			return m.executeDialogOption()
		} else if m.activeTab == SchemaTab && m.schema != nil && !m.showSchemaFilter {
			// Show column query dialog for the selected schema column
			filteredFields := m.getFilteredSchemaFields()
			if m.schemaRowCursor < len(filteredFields) {
				m.selectedColumn = filteredFields[m.schemaRowCursor]
				m.showColumnDialog = true
				m.dialogCursor = 0
				return m, nil
			}
		} else if m.activeTab == QueryTab && !m.queryInput.Focused() {
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

// Shared function to calculate column widths consistently
func (m *TableDetailModel) calculateColumnWidths() []int {
	if len(m.preview.Headers) == 0 {
		return []int{}
	}

	colWidths := make([]int, len(m.preview.Headers))
	minColWidth := 8
	maxColWidth := 30

	// Use filtered rows for width calculation to be consistent with rendering
	filteredRows := m.getFilteredPreviewRows()

	for i, header := range m.preview.Headers {
		width := len(header)
		for _, row := range filteredRows {
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

	return colWidths
}

func (m *TableDetailModel) ensurePreviewColumnVisible() {
	if len(m.preview.Headers) == 0 {
		return
	}

	// Use shared column width calculation
	colWidths := m.calculateColumnWidths()

	// Calculate start position of selected column
	selectedColStart := 0
	for i := 0; i < m.previewColCursor; i++ {
		selectedColStart += colWidths[i] + 1 // +1 for space
	}

	// Calculate end position of selected column
	selectedColEnd := selectedColStart + colWidths[m.previewColCursor]

	// Adjust horizontal offset to keep selected column visible
	maxDisplayWidth := m.width - 2 // Use actual available width

	// Check if the cursor column is already fully visible
	visibleStart := m.horizontalOffset
	visibleEnd := m.horizontalOffset + maxDisplayWidth

	// If column is already fully visible, don't scroll at all
	if selectedColStart >= visibleStart && selectedColEnd <= visibleEnd {
		return
	}

	// Special case: if we're at the first column, ensure horizontal offset is 0
	if m.previewColCursor == 0 {
		m.horizontalOffset = 0
		return
	}

	// Minimal scrolling - only scroll just enough to make the column visible
	if selectedColStart < visibleStart {
		// Column starts before visible area - scroll left just enough to show the start
		m.horizontalOffset = selectedColStart
	} else if selectedColEnd > visibleEnd {
		// Column ends after visible area - scroll right just enough to show the end
		m.horizontalOffset = m.horizontalOffset + (selectedColEnd - visibleEnd)
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

func (m TableDetailModel) ViewWithLoading(loadingSchema, loadingPreview bool) string {
	return m.viewWithLoadingState(loadingSchema, loadingPreview)
}

func (m TableDetailModel) View() string {
	return m.viewWithLoadingState(false, false)
}

func (m TableDetailModel) viewWithLoadingState(loadingSchema, loadingPreview bool) string {
	var content strings.Builder

	tabs := m.renderTabsWithLoading(loadingSchema, loadingPreview)
	content.WriteString(tabs + "\n\n")

	switch m.activeTab {
	case SchemaTab:
		content.WriteString(m.renderSchemaTab())
	case PreviewTab:
		content.WriteString(m.renderPreviewTab())
	case QueryTab:
		content.WriteString(m.renderQueryTab())
	case ResultsTab:
		content.WriteString(m.renderResultsTab())
	}

	return content.String()
}

func (m TableDetailModel) renderTabsWithLoading(loadingSchema, loadingPreview bool) string {
	return m.renderTabsWithState(loadingSchema, loadingPreview)
}

func (m TableDetailModel) renderTabsWithState(loadingSchema, loadingPreview bool) string {
	var tabs []string

	schemaStyle := TabInactiveStyle
	previewStyle := TabInactiveStyle
	queryStyle := TabInactiveStyle
	resultsStyle := TabInactiveStyle

	switch m.activeTab {
	case SchemaTab:
		schemaStyle = TabActiveStyle
	case PreviewTab:
		previewStyle = TabActiveStyle
	case QueryTab:
		queryStyle = TabActiveStyle
	case ResultsTab:
		resultsStyle = TabActiveStyle
	}

	schemaText := "Schema"
	if loadingSchema {
		schemaText += " (Loading...)"
	}
	previewText := "Preview"
	if loadingPreview {
		previewText += " (Loading...)"
	}

	tabs = append(tabs, schemaStyle.Render(schemaText))
	tabs = append(tabs, previewStyle.Render(previewText))
	tabs = append(tabs, queryStyle.Render("Query"))
	tabs = append(tabs, resultsStyle.Render("Results"))

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
	separatorWidth := m.width - 2
	if separatorWidth < 20 {
		separatorWidth = 20
	}
	content.WriteString(strings.Repeat("─", separatorWidth) + "\n")

	visible := 0
	maxVisible := m.getMaxVisibleSchema()
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
		content.WriteString("\n" + HelpStyle.Render("Press / to search columns, ←→ or h/l to scroll horizontally, Enter to query column"))
	}

	// Render dialog if visible
	if m.showColumnDialog {
		content.WriteString("\n\n" + m.renderColumnDialog())
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
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("Table: %s", m.currentTableName)) + "\n")
	}

	// Show preview filter if active
	if m.showPreviewFilter {
		content.WriteString(SearchBoxStyle.Render(fmt.Sprintf("Search rows: %s", m.previewFilter)) + "\n")
	} else if m.previewFilter != "" {
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("Filter: %s (press / to edit, esc to clear)", m.previewFilter)) + "\n")
	}

	// Show visual mode indicator
	if m.visualMode {
		start := min(m.visualStartRow, m.visualEndRow)
		end := max(m.visualStartRow, m.visualEndRow)
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("VISUAL: %d rows selected (%d-%d)", end-start+1, start+1, end+1)) + "\n")
	}

	content.WriteString("\n")

	if len(m.preview.Headers) == 0 {
		content.WriteString(SubtleItemStyle.Render("No data available"))
		return content.String()
	}

	// Use shared column width calculation to ensure consistency with scrolling
	colWidths := m.calculateColumnWidths()

	// Render header with proper alignment and horizontal scrolling
	headerRow := ""
	currentPos := 0
	for i, header := range m.preview.Headers {
		headerText := truncate(header, colWidths[i])
		cellFormatted := HeaderStyle.Render(fmt.Sprintf("%-*s", colWidths[i], headerText)) + " "

		// Check if this column should be visible based on horizontal offset
		cellStart := currentPos
		cellEnd := currentPos + len(cellFormatted)

		if cellEnd > m.horizontalOffset && cellStart < m.horizontalOffset+(m.width-2) {
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
	maxSeparatorWidth := m.width - 2
	if separatorWidth > maxSeparatorWidth {
		separatorWidth = maxSeparatorWidth
	}
	content.WriteString(strings.Repeat("─", separatorWidth) + "\n")

	// Calculate visible rows based on scroll and display using filtered data
	filteredRows := m.getFilteredPreviewRows()
	maxVisible := m.getMaxVisiblePreview()
	startRow := m.scrollOffset
	endRow := startRow + maxVisible
	if endRow > len(filteredRows) {
		endRow = len(filteredRows)
	}

	// Use the actual pane width for horizontal scrolling
	availableWidth := m.width - 2 // Leave small margin for border/padding
	if availableWidth < 20 {
		availableWidth = 20 // Minimum width
	}

	// Render visible rows with visual mode highlighting
	displayRowIdx := 0 // Track the display row index (like schema's i)
	for absoluteRowIdx := startRow; absoluteRowIdx < endRow; absoluteRowIdx++ {
		if absoluteRowIdx >= len(filteredRows) {
			break
		}
		row := filteredRows[absoluteRowIdx]
		currentPos := 0

		rowContent := ""

		for i, cell := range row {
			if i >= len(m.preview.Headers) {
				break
			}

			// Calculate cell position and visibility
			cellWidth := colWidths[i] + 1 // +1 for space
			cellStart := currentPos
			cellEnd := currentPos + cellWidth

			// Skip cells that are completely outside the visible area
			if cellEnd <= m.horizontalOffset || cellStart >= m.horizontalOffset+availableWidth {
				currentPos += cellWidth
				continue
			}

			// Prepare cell content
			cellStr := fmt.Sprintf("%v", cell)
			cellText := truncate(cellStr, colWidths[i])
			cellFormatted := fmt.Sprintf("%-*s", colWidths[i], cellText)

			// Handle partial visibility by trimming the cell content appropriately
			visibleStart := 0
			visibleEnd := len(cellFormatted)

			if cellStart < m.horizontalOffset {
				visibleStart = m.horizontalOffset - cellStart
			}
			if cellEnd > m.horizontalOffset+availableWidth {
				visibleEnd -= (cellEnd - (m.horizontalOffset + availableWidth))
			}

			// Extract visible portion of cell
			if visibleStart < visibleEnd && visibleStart < len(cellFormatted) {
				if visibleEnd > len(cellFormatted) {
					visibleEnd = len(cellFormatted)
				}
				visibleCell := cellFormatted[visibleStart:visibleEnd]

				// Apply single styling based on selection state (use absoluteRowIdx to match cursor logic)
				isVisualSelected := m.visualMode && m.isRowInVisualSelection(absoluteRowIdx)
				isCursorPosition := absoluteRowIdx == m.previewRowCursor && i == m.previewColCursor

				// Apply proper styling - cursor position is absolute index in filtered data
				var styledCell string
				if isCursorPosition {
					// Use the original SelectedItemStyle
					styledCell = SelectedItemStyle.Render(visibleCell)
				} else if isVisualSelected {
					visualStyle := lipgloss.NewStyle().Background(lipgloss.Color("#44475a")).Foreground(lipgloss.Color("#f8f8f2"))
					styledCell = visualStyle.Render(visibleCell)
				} else {
					styledCell = TableCellStyle.Render(visibleCell)
				}

				rowContent += styledCell

				// Add space separator only if we're not at the edge of visibility
				if cellEnd <= m.horizontalOffset+availableWidth {
					rowContent += " "
				}
			}

			currentPos += cellWidth
		}

		// Add the complete row content to the main content
		content.WriteString(rowContent + "\n")
		displayRowIdx++
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
	info := fmt.Sprintf("Rows %d-%d of %d | Cursor[%d,%d] HOffset:%d | (%s)",
		startRow+1, endRow, len(m.preview.Rows), m.previewRowCursor+1, m.previewColCursor+1, m.horizontalOffset, selectedColumnName)
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
		content.WriteString(HelpStyle.Render("Press Enter to edit query, Esc to exit edit mode, Ctrl+Y to copy query") + "\n")
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

func (m TableDetailModel) getMaxVisibleSchema() int {
	// Calculate actual space used by our UI elements within the content area
	tabHeight := 2       // Tab bar + blank line
	titleHeight := 1     // "🏗 Table Schema" header
	tableNameHeight := 1 // Table name line
	headerHeight := 2    // Column headers + separator line
	helpHeight := 2      // Help text at bottom
	paddingHeight := 1   // Some breathing room

	filterHeight := 0
	if m.showSchemaFilter || m.schemaFilter != "" {
		filterHeight = 2 // Filter display + blank
	}

	dialogHeight := 0
	if m.showColumnDialog {
		// Dialog takes: title + column details + options + help + spacing
		dialogHeight = 6 + m.getDialogOptionCount() // Estimated height for dialog
	}

	// Available space for schema rows
	available := m.height - tabHeight - titleHeight - tableNameHeight - headerHeight - helpHeight - filterHeight - dialogHeight - paddingHeight

	if available < 1 {
		available = 1 // Show at least one row
	}
	return available
}

func (m TableDetailModel) getMaxVisibleResults() int {
	// Calculate actual space used by our UI elements within the content area
	tabHeight := 2       // Tab bar + blank line
	titleHeight := 1     // "📊 Query Results" header
	queryInfoHeight := 2 // Query + Rows info lines
	headerHeight := 1    // Column headers
	helpHeight := 2      // Help text at bottom
	paddingHeight := 1   // Some breathing room

	// Available space for result rows
	available := m.height - tabHeight - titleHeight - queryInfoHeight - headerHeight - helpHeight - paddingHeight

	if available < 1 {
		available = 1 // Show at least one row
	}
	return available
}

func (m TableDetailModel) getMaxVisiblePreview() int {
	// Calculate actual space used by our UI elements within the content area
	tabHeight := 2       // Tab bar + blank line
	titleHeight := 1     // "👀 Table Preview" header
	tableNameHeight := 2 // Table name + blank line
	headerHeight := 2    // Column headers + separator
	helpHeight := 2      // Help text at bottom
	paddingHeight := 1   // Some breathing room

	// Available space for data rows
	available := m.height - tabHeight - titleHeight - tableNameHeight - headerHeight - helpHeight - paddingHeight

	if available < 1 {
		available = 1 // Show at least one row
	}
	return available
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

// handleEscapeKey handles ESC key with proper hierarchy
func (m TableDetailModel) handleEscapeKey() (TableDetailModel, tea.Cmd) {
	// Priority 1: Close column dialog if open
	if m.showColumnDialog {
		m.showColumnDialog = false
		m.selectedColumn = nil
		m.dialogCursor = 0
		return m, nil
	}

	// Priority 2: Exit visual mode if active
	if m.visualMode {
		m.visualMode = false
		m.visualStartRow = 0
		m.visualEndRow = 0
		return m, nil
	}

	// Priority 3: Clear active search filters
	if m.activeTab == SchemaTab && m.schemaFilter != "" {
		m.schemaFilter = ""
		m.schemaRowCursor = 0
		return m, nil
	}

	if m.activeTab == PreviewTab && m.previewFilter != "" {
		m.previewFilter = ""
		m.previewRowCursor = 0
		return m, nil
	}

	// Priority 3: Return focus to dataset list (handled by app.go)
	// This will be caught by app.go's escape handling
	return m, nil
}

// getFilteredPreviewRows returns preview rows that match the current filter
func (m TableDetailModel) getFilteredPreviewRows() [][]interface{} {
	if m.preview == nil {
		return nil
	}

	if m.previewFilter == "" {
		return m.preview.Rows
	}

	var filtered [][]interface{}
	filter := strings.ToLower(m.previewFilter)

	for _, row := range m.preview.Rows {
		// Check if any cell in the row matches the filter
		rowMatches := false
		for _, cell := range row {
			cellStr := strings.ToLower(fmt.Sprintf("%v", cell))
			if strings.Contains(cellStr, filter) {
				rowMatches = true
				break
			}
		}

		if rowMatches {
			filtered = append(filtered, row)
		}
	}

	return filtered
}

// Helper functions for visual selection are defined in app.go

// isRowInVisualSelection checks if a row is within the visual selection range
func (m TableDetailModel) isRowInVisualSelection(rowIdx int) bool {
	if !m.visualMode {
		return false
	}

	start := min(m.visualStartRow, m.visualEndRow)
	end := max(m.visualStartRow, m.visualEndRow)

	return rowIdx >= start && rowIdx <= end
}

// forceScrollToColumn immediately calculates and sets horizontal offset to show a specific column
func (m *TableDetailModel) forceScrollToColumn(colIndex int) {
	if m.preview == nil || colIndex < 0 || colIndex >= len(m.preview.Headers) {
		return
	}

	// Use shared column width calculation for consistency
	colWidths := m.calculateColumnWidths()

	// Calculate start position of target column
	targetColStart := 0
	for i := 0; i < colIndex; i++ {
		targetColStart += colWidths[i] + 1 // +1 for space
	}

	// Calculate end position of target column
	targetColEnd := targetColStart + colWidths[colIndex]

	maxDisplayWidth := m.width - 2 // Use actual available width

	if colIndex == 0 {
		// For first column, scroll to beginning
		m.horizontalOffset = 0
	} else if colIndex == len(m.preview.Headers)-1 {
		// For last column, scroll so it's visible on the right
		m.horizontalOffset = targetColEnd - maxDisplayWidth
		if m.horizontalOffset < 0 {
			m.horizontalOffset = 0
		}
	} else {
		// For middle columns, use normal visibility logic
		if targetColStart < m.horizontalOffset {
			m.horizontalOffset = targetColStart
		} else if targetColEnd > m.horizontalOffset+maxDisplayWidth {
			m.horizontalOffset = targetColEnd - maxDisplayWidth
			if m.horizontalOffset < 0 {
				m.horizontalOffset = 0
			}
		}
	}
}

// getDialogOptionCount returns the number of options available for the selected column
func (m TableDetailModel) getDialogOptionCount() int {
	if m.selectedColumn == nil {
		return 0
	}

	options := 3 // select, select non null, select distinct

	// Add count nulls option only for non-required fields
	if !m.selectedColumn.Required {
		options++
	}

	// Add count empty option only for repeated arrays
	if m.selectedColumn.Repeated {
		options++
	}

	return options
}

// executeDialogOption generates and executes the SQL query for the selected dialog option
func (m TableDetailModel) executeDialogOption() (TableDetailModel, tea.Cmd) {
	if m.selectedColumn == nil {
		return m, nil
	}

	query := m.generateQueryForOption(m.dialogCursor)
	if query == "" {
		return m, nil
	}

	// Clear previous results and reset cursors
	m.queryResults = nil
	m.resultsRowCursor = 0
	m.resultsColCursor = 0
	m.visualMode = false

	// Store the query for the Query tab and populate the query input
	m.executedQuery = query
	m.queryInput.SetValue(query)

	// Close the dialog
	m.showColumnDialog = false

	// Switch to Results tab
	m.activeTab = ResultsTab

	// Return command to execute the query
	return m, func() tea.Msg {
		return ExecuteQueryMsg{Query: query}
	}
}

// generateQueryForOption generates the SQL query based on the selected option
func (m TableDetailModel) generateQueryForOption(optionIndex int) string {
	if m.selectedColumn == nil {
		return ""
	}

	// Construct fully qualified table name
	var fullTableName string
	if m.currentProjectID != "" && m.currentDatasetID != "" && m.currentTableName != "" {
		if m.isDatePartitionedTable() {
			// For partitioned tables, extract the base table name and use wildcard syntax
			baseTableName := m.getBaseTableName()
			fullTableName = fmt.Sprintf("`%s.%s.%s*`", m.currentProjectID, m.currentDatasetID, baseTableName)
		} else {
			fullTableName = fmt.Sprintf("`%s.%s.%s`", m.currentProjectID, m.currentDatasetID, m.currentTableName)
		}
	} else {
		// Fallback to just table name if we don't have full info
		fullTableName = fmt.Sprintf("`%s`", m.currentTableName)
	}

	columnName := m.selectedColumn.Name
	options := []string{}

	// Handle partitioned tables - add _TABLE_SUFFIX constraint for date-partitioned tables
	tableConstraint := ""
	if m.isDatePartitionedTable() {
		// For date-partitioned tables, add a recent date constraint to avoid scanning all partitions
		tableConstraint = " WHERE _TABLE_SUFFIX BETWEEN FORMAT_DATE('%Y%m%d', DATE_SUB(CURRENT_DATE(), INTERVAL 7 DAY)) AND FORMAT_DATE('%Y%m%d', CURRENT_DATE())"
	}

	// Basic select
	if tableConstraint != "" {
		options = append(options, fmt.Sprintf("SELECT %s FROM %s%s LIMIT 100", columnName, fullTableName, tableConstraint))
	} else {
		options = append(options, fmt.Sprintf("SELECT %s FROM %s LIMIT 100", columnName, fullTableName))
	}

	// Select non null
	if tableConstraint != "" {
		options = append(options, fmt.Sprintf("SELECT %s FROM %s%s AND %s IS NOT NULL LIMIT 100", columnName, fullTableName, tableConstraint, columnName))
	} else {
		options = append(options, fmt.Sprintf("SELECT %s FROM %s WHERE %s IS NOT NULL LIMIT 100", columnName, fullTableName, columnName))
	}

	// Select distinct
	if tableConstraint != "" {
		options = append(options, fmt.Sprintf("SELECT DISTINCT %s FROM %s%s LIMIT 100", columnName, fullTableName, tableConstraint))
	} else {
		options = append(options, fmt.Sprintf("SELECT DISTINCT %s FROM %s LIMIT 100", columnName, fullTableName))
	}

	// Count nulls (only for non-required fields)
	if !m.selectedColumn.Required {
		if tableConstraint != "" {
			options = append(options, fmt.Sprintf("SELECT COUNT(*) as null_count FROM %s%s AND %s IS NULL", fullTableName, tableConstraint, columnName))
		} else {
			options = append(options, fmt.Sprintf("SELECT COUNT(*) as null_count FROM %s WHERE %s IS NULL", fullTableName, columnName))
		}
	}

	// Count empty (only for repeated arrays)
	if m.selectedColumn.Repeated {
		if tableConstraint != "" {
			options = append(options, fmt.Sprintf("SELECT COUNT(*) as empty_count FROM %s%s AND ARRAY_LENGTH(%s) = 0", fullTableName, tableConstraint, columnName))
		} else {
			options = append(options, fmt.Sprintf("SELECT COUNT(*) as empty_count FROM %s WHERE ARRAY_LENGTH(%s) = 0", fullTableName, columnName))
		}
	}

	if optionIndex >= 0 && optionIndex < len(options) {
		return options[optionIndex]
	}

	return ""
}

// isDatePartitionedTable checks if the current table appears to be date-partitioned
func (m TableDetailModel) isDatePartitionedTable() bool {
	// Check if table name ends with a date pattern (common for partitioned tables)
	// Examples: table_name_20250101, events__2025_03_30, logs_2025_01_15
	tableName := m.currentTableName
	if len(tableName) < 8 {
		return false
	}

	// Check for common date partition patterns
	// Pattern 1: ends with _YYYYMMDD
	if len(tableName) >= 9 && tableName[len(tableName)-9] == '_' {
		suffix := tableName[len(tableName)-8:]
		if isDateString(suffix, "20060102") {
			return true
		}
	}

	// Pattern 2: ends with __YYYY_MM_DD
	if len(tableName) >= 12 && tableName[len(tableName)-12:len(tableName)-10] == "__" {
		suffix := tableName[len(tableName)-10:]
		if isDateString(suffix, "2006_01_02") {
			return true
		}
	}

	// Pattern 3: ends with _YYYY_MM_DD
	if len(tableName) >= 11 && tableName[len(tableName)-11] == '_' {
		suffix := tableName[len(tableName)-10:]
		if isDateString(suffix, "2006_01_02") {
			return true
		}
	}

	return false
}

// getBaseTableName extracts the base table name from a partitioned table
func (m TableDetailModel) getBaseTableName() string {
	tableName := m.currentTableName

	// Pattern 1: table_name_YYYYMMDD -> table_name_
	if len(tableName) >= 9 && tableName[len(tableName)-9] == '_' {
		suffix := tableName[len(tableName)-8:]
		if isDateString(suffix, "20060102") {
			return tableName[:len(tableName)-8]
		}
	}

	// Pattern 2: table_name__YYYY_MM_DD -> table_name__
	if len(tableName) >= 12 && tableName[len(tableName)-12:len(tableName)-10] == "__" {
		suffix := tableName[len(tableName)-10:]
		if isDateString(suffix, "2006_01_02") {
			return tableName[:len(tableName)-10]
		}
	}

	// Pattern 3: table_name_YYYY_MM_DD -> table_name_
	if len(tableName) >= 11 && tableName[len(tableName)-11] == '_' {
		suffix := tableName[len(tableName)-10:]
		if isDateString(suffix, "2006_01_02") {
			return tableName[:len(tableName)-10]
		}
	}

	// If no pattern matches, return the original table name
	return tableName
}

// isDateString checks if a string matches a date format
func isDateString(s, layout string) bool {
	// Simple validation - check if string has right length and format
	if len(s) != len(layout) {
		return false
	}

	// Check basic pattern - all digits in the right places
	for i, char := range layout {
		if char >= '0' && char <= '9' {
			if s[i] < '0' || s[i] > '9' {
				return false
			}
		} else if s[i] != byte(char) {
			return false
		}
	}

	return true
}

// renderResultsTab renders the results of the executed query with preview-like functionality
func (m TableDetailModel) renderResultsTab() string {
	if m.queryResults == nil {
		return SubtleItemStyle.Render("No query results available. Execute a query from the schema column dialog.")
	}

	var content strings.Builder
	content.WriteString(HeaderStyle.Render("📊 Query Results") + "\n")

	if m.executedQuery != "" {
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("Query: %s", m.executedQuery)) + "\n")
	}

	content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("Rows: %d", len(m.queryResults.Rows))) + "\n")

	// Show visual mode indicator
	if m.visualMode {
		start := min(m.visualStartRow, m.visualEndRow)
		end := max(m.visualStartRow, m.visualEndRow)
		content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("VISUAL: %d rows selected (%d-%d)", end-start+1, start+1, end+1)) + "\n")
	}

	content.WriteString("\n")

	if len(m.queryResults.Rows) == 0 {
		content.WriteString(SubtleItemStyle.Render("No data returned from query."))
		return content.String()
	}

	// Render table similar to preview tab
	return content.String() + m.renderQueryResultsTable()
}

// renderQueryResultsTable renders the results table with navigation
func (m TableDetailModel) renderQueryResultsTable() string {
	if m.queryResults == nil || len(m.queryResults.Rows) == 0 {
		return ""
	}

	var content strings.Builder

	// Calculate column widths
	colWidths := make([]int, len(m.queryResults.Columns))
	for i, header := range m.queryResults.Columns {
		colWidths[i] = len(header)
		for _, row := range m.queryResults.Rows {
			if i < len(row) {
				cellValue := fmt.Sprintf("%v", row[i])
				if len(cellValue) > colWidths[i] {
					colWidths[i] = len(cellValue)
				}
			}
		}
		// Cap column width at 30 characters
		if colWidths[i] > 30 {
			colWidths[i] = 30
		}
		// Minimum width of 8
		if colWidths[i] < 8 {
			colWidths[i] = 8
		}
	}

	// Render headers
	var headers []string
	for i, header := range m.queryResults.Columns {
		style := HeaderStyle
		if i == m.resultsColCursor {
			style = SelectedHeaderStyle
		}
		// Pad or truncate header to fit column width
		displayHeader := header
		if len(displayHeader) > colWidths[i] {
			displayHeader = displayHeader[:colWidths[i]-3] + "..."
		}
		headers = append(headers, style.Render(fmt.Sprintf("%-*s", colWidths[i], displayHeader)))
	}
	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, headers...) + "\n")

	// Calculate visible rows based on cursor and height
	maxRows := m.getMaxVisibleResults()
	if maxRows < 1 {
		maxRows = 1
	}

	startRow := m.resultsRowCursor - maxRows/2
	if startRow < 0 {
		startRow = 0
	}
	endRow := startRow + maxRows
	if endRow > len(m.queryResults.Rows) {
		endRow = len(m.queryResults.Rows)
		startRow = endRow - maxRows
		if startRow < 0 {
			startRow = 0
		}
	}

	// Render visible rows
	for rowIdx := startRow; rowIdx < endRow; rowIdx++ {
		row := m.queryResults.Rows[rowIdx]
		var cells []string

		for colIdx, colWidth := range colWidths {
			var cellValue string
			if colIdx < len(row) {
				cellValue = fmt.Sprintf("%v", row[colIdx])
			}

			// Truncate if too long
			if len(cellValue) > colWidth {
				cellValue = cellValue[:colWidth-3] + "..."
			}

			style := ItemStyle
			isVisualSelected := m.visualMode && m.isRowInVisualSelection(rowIdx)
			if rowIdx == m.resultsRowCursor && colIdx == m.resultsColCursor {
				style = SelectedItemStyle
			} else if rowIdx == m.resultsRowCursor {
				style = SelectedRowStyle
			} else if isVisualSelected {
				style = lipgloss.NewStyle().Background(lipgloss.Color("#44475a")).Foreground(lipgloss.Color("#f8f8f2"))
			}

			cells = append(cells, style.Render(fmt.Sprintf("%-*s", colWidth, cellValue)))
		}
		content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cells...) + "\n")
	}

	// Show navigation info
	content.WriteString("\n" + SubtleItemStyle.Render(
		fmt.Sprintf("Row %d/%d, Column %d/%d • Use arrow keys to navigate",
			m.resultsRowCursor+1, len(m.queryResults.Rows),
			m.resultsColCursor+1, len(m.queryResults.Columns))))

	// Show copy help text
	if m.visualMode {
		content.WriteString("\n" + HelpStyle.Render("Press y to copy selected rows, V to exit visual mode"))
	} else {
		content.WriteString("\n" + HelpStyle.Render("Press y to copy selected cell, V to enter visual mode"))
	}

	return content.String()
}

// getDialogOptionName returns the display name for a dialog option
func (m TableDetailModel) getDialogOptionName(optionIndex int) string {
	if m.selectedColumn == nil {
		return ""
	}

	options := []string{"Select", "Select Non Null", "Select Distinct"}

	// Add count nulls option only for non-required fields
	if !m.selectedColumn.Required {
		options = append(options, "Count Nulls")
	}

	// Add count empty option only for repeated arrays
	if m.selectedColumn.Repeated {
		options = append(options, "Count Empty")
	}

	if optionIndex >= 0 && optionIndex < len(options) {
		return options[optionIndex]
	}

	return ""
}

// renderColumnDialog renders the dialog box for column query options
func (m TableDetailModel) renderColumnDialog() string {
	if m.selectedColumn == nil {
		return ""
	}

	var content strings.Builder

	dialogTitle := fmt.Sprintf("Query options for column: %s", m.selectedColumn.Name)
	content.WriteString(HeaderStyle.Render(dialogTitle) + "\n")

	// Show column details
	content.WriteString(SubtleItemStyle.Render(fmt.Sprintf("Type: %s | Mode: %s",
		m.selectedColumn.Type,
		m.getColumnMode(m.selectedColumn))) + "\n\n")

	// Render options
	optionCount := m.getDialogOptionCount()
	for i := 0; i < optionCount; i++ {
		optionName := m.getDialogOptionName(i)
		if i == m.dialogCursor {
			content.WriteString(SelectedItemStyle.Render(fmt.Sprintf("► %s", optionName)) + "\n")
		} else {
			content.WriteString(ItemStyle.Render(fmt.Sprintf("  %s", optionName)) + "\n")
		}
	}

	content.WriteString("\n" + HelpStyle.Render("↑↓ to navigate • Enter to execute • Esc to cancel"))

	return content.String()
}

// getColumnMode returns a human-readable mode string for a column
func (m TableDetailModel) getColumnMode(col *bigquery.Column) string {
	if col.Repeated {
		return "REPEATED"
	}
	if col.Required {
		return "REQUIRED"
	}
	return "NULLABLE"
}

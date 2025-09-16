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
	height           int
	width            int
	// Visual selection mode
	visualMode     bool
	visualStartRow int
	visualEndRow   int
	// Preview filtering
	previewFilter     string
	showPreviewFilter bool
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
		if m.activeTab == PreviewTab && m.preview != nil {
			m.visualMode = !m.visualMode
			if m.visualMode {
				m.visualStartRow = m.previewRowCursor
				m.visualEndRow = m.previewRowCursor
			} else {
				m.visualStartRow = 0
				m.visualEndRow = 0
			}
		}
		return m, nil
	}

	// Handle search trigger
	if msg.String() == "/" {
		if m.activeTab == SchemaTab {
			m.showSchemaFilter = true
			return m, nil
		} else if m.activeTab == PreviewTab {
			m.showPreviewFilter = true
			return m, nil
		}
	}

	// Handle horizontal navigation shortcuts
	if msg.String() == "0" {
		if m.activeTab == PreviewTab {
			m.previewColCursor = 0
			// Force horizontal scroll to show the first column
			m.forceScrollToColumn(0)
		} else if m.activeTab == SchemaTab {
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
		m.activeTab = TabType((int(m.activeTab) + 1) % 3)
		m.scrollOffset = 0
		m.previewRowCursor = 0
		m.previewColCursor = 0
		m.schemaRowCursor = 0
		m.visualMode = false // Exit visual mode when changing tabs

	case key.Matches(msg, DefaultKeyMap().ShiftTab):
		m.activeTab = TabType((int(m.activeTab) + 2) % 3) // +2 is same as -1 in mod 3
		m.scrollOffset = 0
		m.previewRowCursor = 0
		m.previewColCursor = 0
		m.schemaRowCursor = 0
		m.visualMode = false // Exit visual mode when changing tabs

	case key.Matches(msg, DefaultKeyMap().Up):
		if m.activeTab == PreviewTab && m.preview != nil {
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
		} else {
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		}

	case key.Matches(msg, DefaultKeyMap().Down):
		if m.activeTab == PreviewTab && m.preview != nil {
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
	}

	return content.String()
}

func (m TableDetailModel) renderTabsWithLoading(loadingSchema, loadingPreview bool) string {
	return m.renderTabsWithState(loadingSchema, loadingPreview)
}

func (m TableDetailModel) renderTabs() string {
	return m.renderTabsWithState(false, false)
}

func (m TableDetailModel) renderTabsWithState(loadingSchema, loadingPreview bool) string {
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

	// Available space for schema rows
	available := m.height - tabHeight - titleHeight - tableNameHeight - headerHeight - helpHeight - filterHeight - paddingHeight

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
	// Priority 1: Exit visual mode if active
	if m.visualMode {
		m.visualMode = false
		m.visualStartRow = 0
		m.visualEndRow = 0
		return m, nil
	}

	// Priority 2: Clear active search filters
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

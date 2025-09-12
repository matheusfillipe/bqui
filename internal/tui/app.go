package tui

import (
	"context"
	"fmt"
	"strings"

	"bqui/internal/bigquery"
	"bqui/pkg/clipboard"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Enter       key.Binding
	Tab         key.Binding
	ShiftTab    key.Binding
	Search      key.Binding
	Copy        key.Binding
	CopyAlt     key.Binding
	Top         key.Binding
	Bottom      key.Binding
	VimTop      key.Binding
	VimBottom   key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	ProjectList key.Binding
	Escape      key.Binding
	Back        key.Binding
	Quit        key.Binding
	Help        key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "move right"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous tab"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Copy: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy table name"),
		),
		CopyAlt: key.NewBinding(
			key.WithKeys("ctrl+y"),
			key.WithHelp("ctrl+y", "copy table name"),
		),
		Top: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "go to top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "go to bottom"),
		),
		VimTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "go to top"),
		),
		VimBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to bottom"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "page down"),
		),
		ProjectList: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "project selector"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/cancel/clear"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "back to left pane"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Tab, k.ShiftTab, k.Search, k.Copy, k.Quit, k.Help}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Tab, k.ShiftTab, k.Search, k.Escape},
		{k.Copy, k.CopyAlt, k.Top, k.Bottom},
		{k.VimTop, k.VimBottom, k.PageUp, k.PageDown},
		{k.ProjectList, k.Back, k.Quit, k.Help},
	}
}

type FocusState int

const (
	FocusDatasetList FocusState = iota
	FocusTableDetail
	FocusProjectSelector
	FocusSearch
)

type Model struct {
	ctx              context.Context
	bqClient         *bigquery.Client
	datasetList      DatasetListModel
	tableDetail      TableDetailModel
	projectSelector  ProjectSelectorModel
	search           SearchModel
	focus            FocusState
	keyMap           KeyMap
	help             help.Model
	showHelp         bool
	width            int
	height           int
	ready            bool
	err              error
	statusMessage    string
	showProjectList  bool
}

func NewModel(ctx context.Context, bqClient *bigquery.Client) Model {
	m := Model{
		ctx:             ctx,
		bqClient:        bqClient,
		datasetList:     NewDatasetListModel(),
		tableDetail:     NewTableDetailModel(),
		projectSelector: NewProjectSelectorModel(),
		search:          NewSearchModel(),
		focus:           FocusDatasetList,
		keyMap:          DefaultKeyMap(),
		help:            help.New(),
		showHelp:        false,
		ready:           false,
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.datasetList.Init(),
		m.loadDatasets(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		if m.focus == FocusSearch {
			return m.handleSearchInput(msg)
		}

		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keyMap.ProjectList):
			if m.focus != FocusProjectSelector {
				m.focus = FocusProjectSelector
				m.showProjectList = true
				return m, m.loadProjects()
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Escape):
			if m.showProjectList {
				m.showProjectList = false
				m.focus = FocusDatasetList
				return m, nil
			}
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			if m.focus == FocusTableDetail {
				m.focus = FocusDatasetList
				return m, nil
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Tab, m.keyMap.ShiftTab):
			if m.focus == FocusTableDetail {
				// Tab cycles through tabs within the right pane
				return m.updateFocusedComponent(msg)
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Search):
			// If we're focused on table detail and in schema tab, let table detail handle the search
			if m.focus == FocusTableDetail && m.tableDetail.activeTab == SchemaTab {
				return m.updateFocusedComponent(msg)
			}
			// Otherwise, trigger global dataset search
			m.focus = FocusSearch
			m.search = NewSearchModel()
			return m, nil


		case key.Matches(msg, m.keyMap.Copy, m.keyMap.CopyAlt):
			return m.handleCopy()
		}

		return m.updateFocusedComponent(msg)

	case DatasetsLoadedMsg:
		m.datasetList.datasets = msg.Datasets
		m.statusMessage = fmt.Sprintf("Loaded %d datasets", len(msg.Datasets))
		return m, nil

	case TablesLoadedMsg:
		m.datasetList.tables = msg.Tables
		m.statusMessage = fmt.Sprintf("Loaded %d tables", len(msg.Tables))
		return m, nil

	case TableSchemaLoadedMsg:
		m.tableDetail.schema = msg.Schema
		m.tableDetail.currentTableName = msg.TableID
		m.tableDetail.schemaRowCursor = 0 // Reset schema row cursor for new data
		m.statusMessage = fmt.Sprintf("Loaded schema for %s", msg.TableID)
		return m, nil

	case TablePreviewLoadedMsg:
		m.tableDetail.preview = msg.Preview
		if m.tableDetail.currentTableName == "" {
			m.tableDetail.currentTableName = msg.TableID
		}
		m.tableDetail.previewRowCursor = 0 // Reset row cursor for new data
		m.tableDetail.previewColCursor = 0 // Reset column cursor for new data
		m.statusMessage = fmt.Sprintf("Loaded preview for %s", msg.TableID)
		return m, nil

	case ErrorMsg:
		m.err = msg.Error
		m.statusMessage = fmt.Sprintf("Error: %s", msg.Error.Error())
		return m, nil

	case CopySuccessMsg:
		m.statusMessage = fmt.Sprintf("Copied: %s", msg.Text)
		return m, nil

	case ProjectSelectedMsg:
		return m, m.switchProject(msg.Project.ID)

	case ProjectSwitchedMsg:
		m.statusMessage = fmt.Sprintf("Switched to project: %s", msg.ProjectID)
		m.showProjectList = false
		m.focus = FocusDatasetList
		m.datasetList = NewDatasetListModel() // Reset dataset list
		m.tableDetail = NewTableDetailModel() // Reset table detail
		return m, m.loadDatasets()
	}

	return m, cmd
}

func (m Model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.focus = FocusDatasetList
		m.datasetList.filter = m.search.input
		return m, nil
	case "esc":
		m.focus = FocusDatasetList
		m.search = NewSearchModel()
		m.datasetList.filter = ""
		return m, nil
	default:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}
}

func (m Model) updateFocusedComponent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focus {
	case FocusDatasetList:
		m.datasetList, cmd = m.datasetList.Update(msg)
		// Check if user explicitly selected a table (pressed Enter)
		if m.datasetList.tableSelected {
			m.datasetList.tableSelected = false // Reset flag
			m.focus = FocusTableDetail
			return m, tea.Batch(cmd, m.loadTableSchema(), m.loadTablePreview())
		}
		// Load preview when hovering over tables, but don't switch focus
		if m.datasetList.selectedDataset != nil && m.datasetList.selectedTable != nil {
			return m, tea.Batch(cmd, m.loadTableSchema(), m.loadTablePreview())
		}
		if m.datasetList.selectedDataset != nil {
			return m, tea.Batch(cmd, m.loadTables())
		}
		return m, cmd

	case FocusTableDetail:
		m.tableDetail, cmd = m.tableDetail.Update(msg)
		return m, cmd

	case FocusProjectSelector:
		m.projectSelector, cmd = m.projectSelector.Update(msg)
		return m, cmd
	}

	return m, cmd
}

func (m Model) handleCopy() (tea.Model, tea.Cmd) {
	if m.focus == FocusDatasetList && m.datasetList.selectedTable != nil {
		fullTableName := fmt.Sprintf("%s.%s.%s", 
			m.bqClient.GetProjectID(),
			m.datasetList.selectedTable.DatasetID,
			m.datasetList.selectedTable.ID)
		return m, func() tea.Msg {
			if err := clipboard.Copy(fullTableName); err != nil {
				return ErrorMsg{Error: err}
			}
			return CopySuccessMsg{Text: fullTableName}
		}
	}
	
	if m.focus == FocusTableDetail && m.tableDetail.activeTab == PreviewTab && m.tableDetail.preview != nil {
		if len(m.tableDetail.preview.Rows) > m.tableDetail.previewRowCursor {
			row := m.tableDetail.preview.Rows[m.tableDetail.previewRowCursor]
			if len(row) > m.tableDetail.previewColCursor {
				cellValue := fmt.Sprintf("%v", row[m.tableDetail.previewColCursor])
				return m, func() tea.Msg {
					if err := clipboard.Copy(cellValue); err != nil {
						return ErrorMsg{Error: err}
					}
					return CopySuccessMsg{Text: fmt.Sprintf("Copied cell: %s", cellValue)}
				}
			}
		}
	}
	
	if m.focus == FocusTableDetail && m.tableDetail.activeTab == SchemaTab && m.tableDetail.schema != nil {
		filteredFields := m.tableDetail.getFilteredSchemaFields()
		if len(filteredFields) > m.tableDetail.schemaRowCursor {
			fieldName := filteredFields[m.tableDetail.schemaRowCursor].Name
			return m, func() tea.Msg {
				if err := clipboard.Copy(fieldName); err != nil {
					return ErrorMsg{Error: err}
				}
				return CopySuccessMsg{Text: fmt.Sprintf("Copied field name: %s", fieldName)}
			}
		}
	}
	
	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.showProjectList {
		return m.projectSelector.View()
	}

	if m.showHelp {
		return m.renderCustomHelp()
	}

	leftPaneWidth := m.width / 3
	rightPaneWidth := m.width - leftPaneWidth - 4

	leftPaneStyle := PaneStyle.Width(leftPaneWidth).Height(m.height - 6)
	rightPaneStyle := PaneStyle.Width(rightPaneWidth).Height(m.height - 6)

	if m.focus == FocusDatasetList {
		leftPaneStyle = ActivePaneStyle.Width(leftPaneWidth).Height(m.height - 6)
	} else if m.focus == FocusTableDetail {
		rightPaneStyle = ActivePaneStyle.Width(rightPaneWidth).Height(m.height - 6)
	}

	leftPane := leftPaneStyle.Render(m.datasetList.View())
	rightPane := rightPaneStyle.Render(m.tableDetail.View())

	mainView := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	statusBar := m.renderStatusBar()
	searchBar := ""
	if m.focus == FocusSearch {
		searchBar = SearchBoxStyle.Render(fmt.Sprintf("Search: %s", m.search.View()))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainView,
		searchBar,
		statusBar,
	)
}

func (m Model) renderStatusBar() string {
	left := m.statusMessage
	if m.err != nil {
		left = ErrorStyle.Render(fmt.Sprintf("Error: %s", m.err.Error()))
	}

	helpText := "Press ? for help"
	helpStyled := HelpStyle.Render(helpText)
	
	// Check if status message and help text fit on one line
	totalWidth := lipgloss.Width(left) + lipgloss.Width(helpStyled) + 3 // +3 for spacing
	
	if totalWidth <= m.width {
		// Fit on one line
		padding := m.width - lipgloss.Width(left) - lipgloss.Width(helpStyled)
		if padding < 0 {
			padding = 0
		}
		statusLine := left + strings.Repeat(" ", padding) + helpStyled
		return StatusBarStyle.Width(m.width).Render(statusLine)
	} else {
		// Split into two lines
		statusLine := StatusBarStyle.Width(m.width).Render(left)
		helpLine := StatusBarStyle.Width(m.width).Render(helpStyled)
		return statusLine + "\n" + helpLine
	}
}

func (m Model) renderCustomHelp() string {
	var content strings.Builder
	
	content.WriteString(HeaderStyle.Render("🔧 bqui - BigQuery Terminal UI Help") + "\n\n")
	
	content.WriteString(HeaderStyle.Render("Navigation:") + "\n")
	content.WriteString("  ↑↓ or j/k         Move up/down in lists\n")
	content.WriteString("  ←→ or h/l         Horizontal scroll (in schema/preview)\n")
	content.WriteString("  Enter             Select dataset/table\n")
	content.WriteString("  Esc               Go back / cancel\n\n")
	
	content.WriteString(HeaderStyle.Render("Tabs (Right Pane):") + "\n")
	content.WriteString("  Tab               Next tab (Schema → Preview → Query)\n")
	content.WriteString("  Shift+Tab         Previous tab (Query → Preview → Schema)\n\n")
	
	content.WriteString(HeaderStyle.Render("Search & Filter:") + "\n")
	content.WriteString("  /                 Search/filter datasets or columns\n")
	content.WriteString("  Esc               Clear filter\n\n")
	
	content.WriteString(HeaderStyle.Render("Actions:") + "\n")
	content.WriteString("  y or Ctrl+Y       Copy table name to clipboard\n")
	content.WriteString("  Ctrl+P            Switch projects\n\n")
	
	content.WriteString(HeaderStyle.Render("Vim Shortcuts:") + "\n")
	content.WriteString("  g / Home          Go to top\n")
	content.WriteString("  G / End           Go to bottom\n")
	content.WriteString("  Page Up/Down      Page navigation\n\n")
	
	content.WriteString(HeaderStyle.Render("Other:") + "\n")
	content.WriteString("  ?                 Show/hide this help\n")
	content.WriteString("  q or Ctrl+C       Quit\n\n")
	
	content.WriteString(HelpStyle.Render("Press any key to close help"))
	
	return content.String()
}
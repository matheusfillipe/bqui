package tui

import (
	"fmt"

	"bqui/internal/bigquery"

	tea "github.com/charmbracelet/bubbletea"
)

type DatasetsLoadedMsg struct {
	Datasets []*bigquery.Dataset
}

type TablesLoadedMsg struct {
	Tables []*bigquery.Table
}

type TableSchemaLoadedMsg struct {
	TableID string
	Schema  *bigquery.TableSchema
}

type TablePreviewLoadedMsg struct {
	TableID string
	Preview *bigquery.TablePreview
}

type QueryResultMsg struct {
	Result *bigquery.QueryResult
}

type ProjectsLoadedMsg struct {
	Projects []*bigquery.Project
}

type ErrorMsg struct {
	Error error
}

type CopySuccessMsg struct {
	Text string
}

type ProjectSwitchedMsg struct {
	ProjectID string
}

type ProjectSelectedMsg struct {
	Project *bigquery.Project
}

func (m Model) loadDatasets() tea.Cmd {
	return func() tea.Msg {
		datasets, err := m.bqClient.ListDatasets()
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to load datasets: %w", err)}
		}
		return DatasetsLoadedMsg{Datasets: datasets}
	}
}

func (m Model) loadTables() tea.Cmd {
	if m.datasetList.selectedDataset == nil {
		return nil
	}

	datasetID := m.datasetList.selectedDataset.ID
	return func() tea.Msg {
		tables, err := m.bqClient.ListTables(datasetID)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to load tables for dataset %s: %w", datasetID, err)}
		}
		return TablesLoadedMsg{Tables: tables}
	}
}

func (m Model) loadTableSchema() tea.Cmd {
	if m.datasetList.selectedTable == nil {
		return nil
	}

	table := m.datasetList.selectedTable
	return func() tea.Msg {
		schema, err := m.bqClient.GetTableSchema(table.DatasetID, table.ID)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to load schema for table %s: %w", table.ID, err)}
		}
		return TableSchemaLoadedMsg{
			TableID: table.ID,
			Schema:  schema,
		}
	}
}

func (m Model) loadTablePreview() tea.Cmd {
	if m.datasetList.selectedTable == nil {
		return nil
	}

	table := m.datasetList.selectedTable
	return func() tea.Msg {
		preview, err := m.bqClient.PreviewTable(table.DatasetID, table.ID, 100)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to load preview for table %s: %w", table.ID, err)}
		}
		return TablePreviewLoadedMsg{
			TableID: table.ID,
			Preview: preview,
		}
	}
}

func (m Model) executeQuery(query string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.bqClient.ExecuteQuery(query)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to execute query: %w", err)}
		}
		return QueryResultMsg{Result: result}
	}
}

func (m Model) loadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := m.bqClient.ListProjects()
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to load projects: %w", err)}
		}
		return ProjectsLoadedMsg{Projects: projects}
	}
}

func (m Model) switchProject(projectID string) tea.Cmd {
	return func() tea.Msg {
		err := m.bqClient.SwitchProject(projectID)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to switch to project %s: %w", projectID, err)}
		}
		return ProjectSwitchedMsg{ProjectID: projectID}
	}
}
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
	DatasetID string
	Tables    []*bigquery.Table
}

type TableSchemaLoadedMsg struct {
	DatasetID string
	TableID   string
	Schema    *bigquery.TableSchema
}

type TablePreviewLoadedMsg struct {
	DatasetID string
	TableID   string
	Preview   *bigquery.TablePreview
}

type QueryResultMsg struct {
	Result *bigquery.QueryResult
}

type ExecuteQueryMsg struct {
	Query string
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
		projectID := m.bqClient.GetProjectID()

		// Try cache first if available
		if m.cache != nil {
			if cachedDatasets, found := m.cache.GetDatasets(projectID); found {
				return DatasetsLoadedMsg{Datasets: cachedDatasets}
			}
		}

		// Load from BigQuery
		datasets, err := m.bqClient.ListDatasets()
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to load datasets: %w", err)}
		}

		// Cache the result if cache is available
		if m.cache != nil {
			if err := m.cache.SetDatasets(projectID, datasets); err != nil {
				// Continue even if caching fails, just log internally
				// In a real app, you might want to log this error
			}
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
		projectID := m.bqClient.GetProjectID()

		// Try cache first if available
		if m.cache != nil {
			if cachedTables, found := m.cache.GetTables(projectID, datasetID); found {
				return TablesLoadedMsg{DatasetID: datasetID, Tables: cachedTables}
			}
		}

		// Load from BigQuery
		tables, err := m.bqClient.ListTables(datasetID)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to load tables for dataset %s: %w", datasetID, err)}
		}

		// Cache the result if cache is available
		if m.cache != nil {
			if err := m.cache.SetTables(projectID, datasetID, tables); err != nil {
				// Continue even if caching fails
			}
		}

		return TablesLoadedMsg{DatasetID: datasetID, Tables: tables}
	}
}

func (m Model) loadTableSchema() tea.Cmd {
	if m.datasetList.selectedTable == nil {
		return nil
	}

	table := m.datasetList.selectedTable
	return func() tea.Msg {
		projectID := m.bqClient.GetProjectID()

		// Try cache first if available
		if m.cache != nil {
			if cachedSchema, found := m.cache.GetSchema(projectID, table.DatasetID, table.ID); found {
				return TableSchemaLoadedMsg{
					DatasetID: table.DatasetID,
					TableID:   table.ID,
					Schema:    cachedSchema,
				}
			}
		}

		// Load from BigQuery
		schema, err := m.bqClient.GetTableSchema(table.DatasetID, table.ID)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to load schema for table %s: %w", table.ID, err)}
		}

		// Cache the result if cache is available
		if m.cache != nil {
			if err := m.cache.SetSchema(projectID, table.DatasetID, table.ID, schema); err != nil {
				// Continue even if caching fails
			}
		}

		return TableSchemaLoadedMsg{
			DatasetID: table.DatasetID,
			TableID:   table.ID,
			Schema:    schema,
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
			DatasetID: table.DatasetID,
			TableID:   table.ID,
			Preview:   preview,
		}
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

func (m Model) executeQuery(query string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.bqClient.ExecuteQuery(query)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to execute query: %w", err)}
		}
		return QueryResultMsg{Result: result}
	}
}

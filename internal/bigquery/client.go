package bigquery

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Client struct {
	bqClient  *bigquery.Client
	projectID string
	ctx       context.Context
	opts      []option.ClientOption
}

func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*Client, error) {
	bqClient, err := bigquery.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}

	return &Client{
		bqClient:  bqClient,
		projectID: projectID,
		ctx:       ctx,
		opts:      opts,
	}, nil
}

func (c *Client) Close() error {
	return c.bqClient.Close()
}

func (c *Client) GetProjectID() string {
	return c.projectID
}

func (c *Client) ListDatasets() ([]*Dataset, error) {
	datasets := make([]*Dataset, 0)
	it := c.bqClient.Datasets(c.ctx)

	for {
		dataset, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate datasets: %w", err)
		}

		metadata, err := dataset.Metadata(c.ctx)
		if err != nil {
			continue
		}

		datasets = append(datasets, &Dataset{
			ID:          dataset.DatasetID,
			ProjectID:   c.projectID,
			Location:    metadata.Location,
			Description: metadata.Description,
			CreatedAt:   metadata.CreationTime,
			Labels:      metadata.Labels,
		})
	}

	sort.Slice(datasets, func(i, j int) bool {
		return datasets[i].ID < datasets[j].ID
	})

	return datasets, nil
}

func (c *Client) ListTables(datasetID string) ([]*Table, error) {
	dataset := c.bqClient.Dataset(datasetID)
	tables := make([]*Table, 0)
	it := dataset.Tables(c.ctx)

	for {
		table, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate tables: %w", err)
		}

		metadata, err := table.Metadata(c.ctx)
		if err != nil {
			continue
		}

		tables = append(tables, &Table{
			ID:          table.TableID,
			DatasetID:   datasetID,
			ProjectID:   c.projectID,
			Description: metadata.Description,
			CreatedAt:   metadata.CreationTime,
			NumRows:     metadata.NumRows,
			NumBytes:    metadata.NumBytes,
			Type:        string(metadata.Type),
			Labels:      metadata.Labels,
		})
	}

	sort.Slice(tables, func(i, j int) bool {
		return tables[i].ID < tables[j].ID
	})

	return tables, nil
}

func (c *Client) GetTableSchema(datasetID, tableID string) (*TableSchema, error) {
	table := c.bqClient.Dataset(datasetID).Table(tableID)
	metadata, err := table.Metadata(c.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get table metadata: %w", err)
	}

	return &TableSchema{
		Fields: convertBigQuerySchema(metadata.Schema),
	}, nil
}

func (c *Client) PreviewTable(datasetID, tableID string, limit int) (*TablePreview, error) {
	if limit <= 0 {
		limit = 100
	}

	schema, err := c.GetTableSchema(datasetID, tableID)
	if err != nil {
		return nil, err
	}

	// Check if table is partitioned and requires partition filter
	partitionFilter, err := c.getPartitionFilter(datasetID, tableID)
	if err != nil {
		return nil, fmt.Errorf("failed to check partition info: %w", err)
	}

	var query string
	if partitionFilter != "" {
		query = fmt.Sprintf("SELECT * FROM `%s.%s.%s` %s LIMIT %d", c.projectID, datasetID, tableID, partitionFilter, limit)
	} else {
		query = fmt.Sprintf("SELECT * FROM `%s.%s.%s` LIMIT %d", c.projectID, datasetID, tableID, limit)
	}

	q := c.bqClient.Query(query)
	q.UseStandardSQL = true

	it, err := q.Read(c.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute preview query: %w", err)
	}

	var rows [][]interface{}
	var headers []string

	for i, field := range schema.Fields {
		headers = append(headers, field.Name)
		if i > 10 {
			headers = append(headers, "...")
			break
		}
	}

	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}

		rowData := make([]interface{}, len(headers))
		for i, val := range row {
			if i >= len(headers) {
				break
			}
			rowData[i] = val
		}
		rows = append(rows, rowData)

		if len(rows) >= limit {
			break
		}
	}

	return &TablePreview{
		Schema:  schema,
		Rows:    rows,
		Headers: headers,
	}, nil
}

func (c *Client) ExecuteQuery(query string) (*QueryResult, error) {
	q := c.bqClient.Query(query)
	q.UseStandardSQL = true

	job, err := q.Run(c.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run query: %w", err)
	}

	status, err := job.Wait(c.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for job completion: %w", err)
	}

	if status.Err() != nil {
		return nil, fmt.Errorf("query failed: %w", status.Err())
	}

	it, err := job.Read(c.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read query results: %w", err)
	}

	var columns []string
	var rows [][]interface{}

	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate query results: %w", err)
		}

		if len(columns) == 0 {
			for i := range row {
				columns = append(columns, fmt.Sprintf("col_%d", i))
			}
		}

		rowData := make([]interface{}, len(row))
		for i, val := range row {
			rowData[i] = val
		}
		rows = append(rows, rowData)
	}

	return &QueryResult{
		Columns: columns,
		Rows:    rows,
		JobID:   job.ID(),
	}, nil
}

func convertBigQuerySchema(schema bigquery.Schema) []*Column {
	var fields []*Column
	for _, field := range schema {
		fields = append(fields, convertBigQueryField(field))
	}
	return fields
}

func convertBigQueryField(field *bigquery.FieldSchema) *Column {
	column := &Column{
		Name:        field.Name,
		Type:        field.Type,
		Repeated:    field.Repeated,
		Required:    field.Required,
		Description: field.Description,
	}

	if len(field.Schema) > 0 {
		for _, subField := range field.Schema {
			column.Fields = append(column.Fields, convertBigQueryField(subField))
		}
	}

	return column
}

func (c *Client) ListProjects() ([]*Project, error) {
	cmd := exec.Command("gcloud", "projects", "list", "--format=value(projectId,name)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects using gcloud: %w", err)
	}

	var projects []*Project
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 1 {
			project := &Project{
				ID:   parts[0],
				Name: parts[0],
			}
			if len(parts) > 1 {
				project.Name = strings.Join(parts[1:], " ")
			}
			projects = append(projects, project)
		}
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].ID < projects[j].ID
	})

	return projects, nil
}

// getPartitionFilter checks if a table is partitioned and returns an appropriate WHERE clause
func (c *Client) getPartitionFilter(datasetID, tableID string) (string, error) {
	table := c.bqClient.Dataset(datasetID).Table(tableID)
	metadata, err := table.Metadata(c.ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get table metadata: %w", err)
	}

	// Check for time partitioning
	if metadata.TimePartitioning != nil {
		if metadata.RequirePartitionFilter {
			// If no field is specified, it's ingestion-time partitioning
			if metadata.TimePartitioning.Field == "" {
				// Use _PARTITIONTIME with a broad date range to ensure results
				return "WHERE _PARTITIONTIME >= TIMESTAMP('1970-01-01')", nil
			} else {
				// Use the specific partition column with a broad range
				return fmt.Sprintf("WHERE %s >= TIMESTAMP('1970-01-01')", metadata.TimePartitioning.Field), nil
			}
		}
	}

	// Check for range partitioning
	if metadata.RangePartitioning != nil {
		if metadata.RequirePartitionFilter {
			// For range partitioning, use a broad numeric range
			return fmt.Sprintf("WHERE %s >= 0", metadata.RangePartitioning.Field), nil
		}
	}

	// Check for common partitioning patterns by examining column names
	partitionFilter, err := c.detectPartitionColumns(datasetID, tableID)
	if err != nil {
		// If we can't detect via INFORMATION_SCHEMA, continue without filter
		return "", nil
	}

	return partitionFilter, nil
}

// detectPartitionColumns checks for common partition columns using INFORMATION_SCHEMA
func (c *Client) detectPartitionColumns(datasetID, tableID string) (string, error) {
	// Common partition column patterns
	commonPartitionColumns := []string{
		"_PARTITIONTIME", "_PARTITIONDATE", "_PARTITION_LOAD_TIME",
		"date", "created_at", "event_date", "transaction_date", "partition_date", "dt",
	}

	query := fmt.Sprintf(`
		SELECT column_name, data_type
		FROM `+"`"+`%s.%s.INFORMATION_SCHEMA.COLUMNS`+"`"+`
		WHERE table_name = '%s'
		AND (is_partitioning_column = "YES" OR column_name IN UNNEST(@partition_columns))
		ORDER BY ordinal_position
		LIMIT 1`, c.projectID, datasetID, tableID)

	q := c.bqClient.Query(query)
	q.UseStandardSQL = true
	q.Parameters = []bigquery.QueryParameter{
		{Name: "partition_columns", Value: commonPartitionColumns},
	}

	it, err := q.Read(c.ctx)
	if err != nil {
		// INFORMATION_SCHEMA might not be accessible, return no filter
		return "", nil
	}

	var row []bigquery.Value
	err = it.Next(&row)
	if err == iterator.Done {
		// No partition columns found
		return "", nil
	}
	if err != nil {
		return "", nil
	}

	columnName := row[0].(string)
	dataType := row[1].(string)

	// Create appropriate filter based on data type
	switch {
	case strings.Contains(dataType, "TIMESTAMP"):
		return fmt.Sprintf("WHERE %s >= TIMESTAMP('1970-01-01')", columnName), nil
	case strings.Contains(dataType, "DATE"):
		return fmt.Sprintf("WHERE %s >= DATE('1970-01-01')", columnName), nil
	case strings.Contains(dataType, "INT") || strings.Contains(dataType, "NUMERIC"):
		return fmt.Sprintf("WHERE %s >= 0", columnName), nil
	default:
		// Unknown type, try timestamp filter
		return fmt.Sprintf("WHERE %s >= TIMESTAMP('1970-01-01')", columnName), nil
	}
}

func (c *Client) SwitchProject(projectID string) error {
	if err := c.bqClient.Close(); err != nil {
		return fmt.Errorf("failed to close current client: %w", err)
	}

	newClient, err := bigquery.NewClient(c.ctx, projectID, c.opts...)
	if err != nil {
		return fmt.Errorf("failed to create client for project %s: %w", projectID, err)
	}

	c.bqClient = newClient
	c.projectID = projectID

	return nil
}

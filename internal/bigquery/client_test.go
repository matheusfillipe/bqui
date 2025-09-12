package bigquery

import (
	"context"
	"testing"

	"github.com/goccy/bigquery-emulator/server"
	"github.com/goccy/bigquery-emulator/types"
	"google.golang.org/api/option"
)

func setupTestServer() (*server.Server, string, func(), error) {
	testServer, err := server.New(server.TempStorage)
	if err != nil {
		return nil, "", nil, err
	}

	if err := testServer.Load(
		server.StructSource(
			types.NewProject("test-project",
				types.NewDataset("test_dataset",
					types.NewTable("test_table",
						[]*types.Column{
							{Name: "id", Type: types.INTEGER},
							{Name: "name", Type: types.STRING},
							{Name: "created_at", Type: types.TIMESTAMP},
						},
						types.Data{
							{
								"id":         1,
								"name":       "test_user_1",
								"created_at": "2023-01-01 00:00:00",
							},
							{
								"id":         2,
								"name":       "test_user_2",
								"created_at": "2023-01-02 00:00:00",
							},
						},
					),
				),
			),
		),
	); err != nil {
		return nil, "", nil, err
	}

	if err := testServer.Start(); err != nil {
		return nil, "", nil, err
	}

	endpoint := testServer.URL()
	cleanup := func() {
		testServer.Stop()
	}

	return testServer, endpoint, cleanup, nil
}

func TestNewClient(t *testing.T) {
	_, endpoint, cleanup, err := setupTestServer()
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	client, err := NewClient(ctx, "test-project", 
		option.WithEndpoint(endpoint), 
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	if client.GetProjectID() != "test-project" {
		t.Errorf("Expected project ID 'test-project', got '%s'", client.GetProjectID())
	}
}

func TestListDatasets(t *testing.T) {
	_, endpoint, cleanup, err := setupTestServer()
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	client, err := NewClient(ctx, "test-project",
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	datasets, err := client.ListDatasets()
	if err != nil {
		t.Fatalf("Failed to list datasets: %v", err)
	}

	if len(datasets) != 1 {
		t.Errorf("Expected 1 dataset, got %d", len(datasets))
	}

	if datasets[0].ID != "test_dataset" {
		t.Errorf("Expected dataset ID 'test_dataset', got '%s'", datasets[0].ID)
	}
}

func TestListTables(t *testing.T) {
	_, endpoint, cleanup, err := setupTestServer()
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	client, err := NewClient(ctx, "test-project",
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	tables, err := client.ListTables("test_dataset")
	if err != nil {
		t.Fatalf("Failed to list tables: %v", err)
	}

	if len(tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(tables))
	}

	if tables[0].ID != "test_table" {
		t.Errorf("Expected table ID 'test_table', got '%s'", tables[0].ID)
	}
}

func TestGetTableSchema(t *testing.T) {
	_, endpoint, cleanup, err := setupTestServer()
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	client, err := NewClient(ctx, "test-project",
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	schema, err := client.GetTableSchema("test_dataset", "test_table")
	if err != nil {
		t.Fatalf("Failed to get table schema: %v", err)
	}

	if len(schema.Fields) != 3 {
		t.Errorf("Expected 3 schema fields, got %d", len(schema.Fields))
	}

	expectedFields := []string{"id", "name", "created_at"}
	for i, field := range schema.Fields {
		if field.Name != expectedFields[i] {
			t.Errorf("Expected field name '%s', got '%s'", expectedFields[i], field.Name)
		}
	}
}

func TestPreviewTable(t *testing.T) {
	_, endpoint, cleanup, err := setupTestServer()
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer cleanup()

	ctx := context.Background()
	client, err := NewClient(ctx, "test-project",
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	preview, err := client.PreviewTable("test_dataset", "test_table", 10)
	if err != nil {
		t.Fatalf("Failed to preview table: %v", err)
	}

	if len(preview.Headers) != 3 {
		t.Errorf("Expected 3 headers, got %d", len(preview.Headers))
	}

	if len(preview.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(preview.Rows))
	}
}
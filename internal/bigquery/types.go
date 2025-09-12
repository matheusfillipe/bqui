package bigquery

import (
	"time"

	"cloud.google.com/go/bigquery"
)

type Project struct {
	ID   string
	Name string
}

type Dataset struct {
	ID          string
	ProjectID   string
	Location    string
	Description string
	CreatedAt   time.Time
	Labels      map[string]string
}

type Table struct {
	ID          string
	DatasetID   string
	ProjectID   string
	Description string
	CreatedAt   time.Time
	NumRows     uint64
	NumBytes    int64
	Type        string
	Labels      map[string]string
}

type Column struct {
	Name        string
	Type        bigquery.FieldType
	Repeated    bool
	Required    bool
	Description string
	Fields      []*Column
}

type TableSchema struct {
	Fields []*Column
}

type QueryResult struct {
	Columns []string
	Rows    [][]interface{}
	JobID   string
}

type TablePreview struct {
	Schema  *TableSchema
	Rows    [][]interface{}
	Headers []string
}
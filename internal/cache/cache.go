package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"bqui/internal/bigquery"
)

type Cache struct {
	baseDir string
}

// New creates a new cache instance with OS-appropriate cache directory
func New() (*Cache, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}

	// Create bqui cache subdirectory
	bquiCacheDir := filepath.Join(cacheDir, "bqui")
	if err := os.MkdirAll(bquiCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &Cache{baseDir: bquiCacheDir}, nil
}

// getCacheDir returns the appropriate cache directory for the OS
func getCacheDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		cacheDir := os.Getenv("LOCALAPPDATA")
		if cacheDir == "" {
			cacheDir = os.Getenv("TEMP")
		}
		if cacheDir == "" {
			return "", fmt.Errorf("cannot determine cache directory on Windows")
		}
		return cacheDir, nil
	case "darwin":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, "Library", "Caches"), nil
	default: // Linux and other Unix-like systems
		cacheDir := os.Getenv("XDG_CACHE_HOME")
		if cacheDir != "" {
			return cacheDir, nil
		}
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".cache"), nil
	}
}

// CachedDatasets represents cached dataset information
type CachedDatasets struct {
	Datasets  []*bigquery.Dataset `json:"datasets"`
	ProjectID string              `json:"project_id"`
	CachedAt  time.Time           `json:"cached_at"`
}

// CachedTables represents cached table information for a dataset
type CachedTables struct {
	Tables    []*bigquery.Table `json:"tables"`
	DatasetID string            `json:"dataset_id"`
	ProjectID string            `json:"project_id"`
	CachedAt  time.Time         `json:"cached_at"`
}

// CachedSchema represents cached schema information for a table
type CachedSchema struct {
	Schema    *bigquery.TableSchema `json:"schema"`
	TableID   string                `json:"table_id"`
	DatasetID string                `json:"dataset_id"`
	ProjectID string                `json:"project_id"`
	CachedAt  time.Time             `json:"cached_at"`
}

// GetDatasets retrieves cached datasets for a project
func (c *Cache) GetDatasets(projectID string) ([]*bigquery.Dataset, bool) {
	filename := filepath.Join(c.baseDir, fmt.Sprintf("datasets_%s.json", projectID))

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, false
	}

	var cached CachedDatasets
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}

	// Check if cache is still valid (24 hours)
	if time.Since(cached.CachedAt) > 24*time.Hour {
		return nil, false
	}

	return cached.Datasets, true
}

// SetDatasets caches datasets for a project
func (c *Cache) SetDatasets(projectID string, datasets []*bigquery.Dataset) error {
	cached := CachedDatasets{
		Datasets:  datasets,
		ProjectID: projectID,
		CachedAt:  time.Now(),
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal datasets: %w", err)
	}

	filename := filepath.Join(c.baseDir, fmt.Sprintf("datasets_%s.json", projectID))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write datasets cache: %w", err)
	}

	return nil
}

// GetTables retrieves cached tables for a dataset
func (c *Cache) GetTables(projectID, datasetID string) ([]*bigquery.Table, bool) {
	filename := filepath.Join(c.baseDir, fmt.Sprintf("tables_%s_%s.json", projectID, datasetID))

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, false
	}

	var cached CachedTables
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}

	// Check if cache is still valid (24 hours)
	if time.Since(cached.CachedAt) > 24*time.Hour {
		return nil, false
	}

	return cached.Tables, true
}

// SetTables caches tables for a dataset
func (c *Cache) SetTables(projectID, datasetID string, tables []*bigquery.Table) error {
	cached := CachedTables{
		Tables:    tables,
		DatasetID: datasetID,
		ProjectID: projectID,
		CachedAt:  time.Now(),
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tables: %w", err)
	}

	filename := filepath.Join(c.baseDir, fmt.Sprintf("tables_%s_%s.json", projectID, datasetID))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write tables cache: %w", err)
	}

	return nil
}

// GetSchema retrieves cached schema for a table
func (c *Cache) GetSchema(projectID, datasetID, tableID string) (*bigquery.TableSchema, bool) {
	filename := filepath.Join(c.baseDir, fmt.Sprintf("schema_%s_%s_%s.json", projectID, datasetID, tableID))

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, false
	}

	var cached CachedSchema
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}

	// Check if cache is still valid (24 hours)
	if time.Since(cached.CachedAt) > 24*time.Hour {
		return nil, false
	}

	return cached.Schema, true
}

// SetSchema caches schema for a table
func (c *Cache) SetSchema(projectID, datasetID, tableID string, schema *bigquery.TableSchema) error {
	cached := CachedSchema{
		Schema:    schema,
		TableID:   tableID,
		DatasetID: datasetID,
		ProjectID: projectID,
		CachedAt:  time.Now(),
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	filename := filepath.Join(c.baseDir, fmt.Sprintf("schema_%s_%s_%s.json", projectID, datasetID, tableID))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write schema cache: %w", err)
	}

	return nil
}

// ClearDatasets removes cached datasets for a project
func (c *Cache) ClearDatasets(projectID string) error {
	filename := filepath.Join(c.baseDir, fmt.Sprintf("datasets_%s.json", projectID))
	err := os.Remove(filename)
	if os.IsNotExist(err) {
		return nil // Not an error if file doesn't exist
	}
	return err
}

// ClearTables removes cached tables for a dataset
func (c *Cache) ClearTables(projectID, datasetID string) error {
	filename := filepath.Join(c.baseDir, fmt.Sprintf("tables_%s_%s.json", projectID, datasetID))
	err := os.Remove(filename)
	if os.IsNotExist(err) {
		return nil // Not an error if file doesn't exist
	}
	return err
}

// ClearSchema removes cached schema for a table
func (c *Cache) ClearSchema(projectID, datasetID, tableID string) error {
	filename := filepath.Join(c.baseDir, fmt.Sprintf("schema_%s_%s_%s.json", projectID, datasetID, tableID))
	err := os.Remove(filename)
	if os.IsNotExist(err) {
		return nil // Not an error if file doesn't exist
	}
	return err
}

// ClearAllTablesInDataset removes all cached tables and schemas for a dataset
func (c *Cache) ClearAllTablesInDataset(projectID, datasetID string) error {
	// Clear tables cache
	if err := c.ClearTables(projectID, datasetID); err != nil {
		return err
	}

	// Clear all schemas in this dataset
	pattern := fmt.Sprintf("schema_%s_%s_*.json", projectID, datasetID)
	matches, err := filepath.Glob(filepath.Join(c.baseDir, pattern))
	if err != nil {
		return err
	}

	for _, match := range matches {
		if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

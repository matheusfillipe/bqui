package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"bqui/internal/bigquery"
	"bqui/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/api/option"
)

var (
	projectID = flag.String("project", "", "BigQuery project ID (if not provided, will use default from credentials)")
	credFile  = flag.String("credentials", "", "Path to service account credentials file (optional)")
	emulator  = flag.String("emulator", "", "BigQuery emulator endpoint (for testing)")
	version   = flag.Bool("version", false, "Show version information")
)

const (
	appVersion = "0.1.0"
	appName    = "bqui"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s\n", appName, appVersion)
		fmt.Println("A BigQuery Terminal User Interface")
		os.Exit(0)
	}

	ctx := context.Background()

	client, err := createBigQueryClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create BigQuery client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing client: %v\n", err)
		}
	}()

	model := tui.NewModel(ctx, client)

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := program.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}

func createBigQueryClient(ctx context.Context) (*bigquery.Client, error) {
	var opts []option.ClientOption

	if *credFile != "" {
		if _, err := os.Stat(*credFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("credentials file not found: %s", *credFile)
		}
		opts = append(opts, option.WithCredentialsFile(*credFile))
	}

	if *emulator != "" {
		opts = append(opts, option.WithEndpoint(*emulator))
		opts = append(opts, option.WithoutAuthentication())
	}

	projID := *projectID
	if projID == "" {
		projID = detectDefaultProject()
		if projID == "" {
			return nil, fmt.Errorf("no project found. Please run 'gcloud config set project PROJECT_ID' or use -project flag")
		}
	}

	return bigquery.NewClient(ctx, projID, opts...)
}

func detectDefaultProject() string {
	// Try environment variables first
	if projID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projID != "" {
		return projID
	}
	if projID := os.Getenv("GCP_PROJECT"); projID != "" {
		return projID
	}

	// Try to get from gcloud config
	if projID := getGCloudDefaultProject(); projID != "" {
		return projID
	}

	return ""
}

func getGCloudDefaultProject() string {
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	projectID := strings.TrimSpace(string(output))
	if projectID == "(unset)" || projectID == "" {
		return ""
	}

	return projectID
}

func init() {
	flag.Usage = func() {
		fmt.Printf("%s - A BigQuery Terminal User Interface\n\n", appName)
		fmt.Printf("Usage: %s [options]\n\n", appName)
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Environment Variables:")
		fmt.Println("  GOOGLE_APPLICATION_CREDENTIALS  Path to service account key file")
		fmt.Println("  GOOGLE_CLOUD_PROJECT             Default project ID")
		fmt.Println("  GCP_PROJECT                      Alternative project ID variable")
		fmt.Println()
		fmt.Println("Key Bindings:")
		fmt.Println("  Navigation:    ↑↓←→ or hjkl")
		fmt.Println("  Select:        Enter")
		fmt.Println("  Search:        /")
		fmt.Println("  Copy table:    y or Ctrl+Y")
		fmt.Println("  Cycle tabs:    Tab")
		fmt.Println("  Back:          Esc")
		fmt.Println("  Project list:  Ctrl+P")
		fmt.Println("  Help:          ?")
		fmt.Println("  Quit:          q or Ctrl+C")
		fmt.Println()
		fmt.Printf("Version: %s\n", appVersion)
	}
}

# bqui 📊

A beautiful Terminal User Interface for Google BigQuery, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

**bqui** makes exploring BigQuery datasets and tables as simple and delightful as using your favorite CLI tools like `lazydocker` or the BigQuery web console.

## ✨ Features

- **🏗️ Intuitive Two-Pane Layout**: Browse datasets and tables in the left pane, view schemas and data in the right pane
- **⌨️ Vim Key Bindings**: Full vim-style navigation (`hjkl`, `g`/`G`, etc.) plus standard arrow keys
- **🔍 Powerful Search**: Filter datasets and tables with `/` - type to search, `Esc` to clear
- **📋 Smart Copy**: Copy full table names to clipboard with `y` or `Ctrl+Y`
- **📊 Schema Viewer**: Inspect table schemas with field types, modes (REQUIRED/REPEATED), and descriptions
- **👀 Data Preview**: Sample table data right in your terminal
- **🔄 Tab Navigation**: Switch between Schema, Preview, and Query tabs with `Tab`
- **🚀 Project Switching**: Access multiple GCP projects with `Ctrl+P`
- **🎨 Beautiful Styling**: Clean, colorful interface with proper syntax highlighting

## 🚀 Installation

### From Source

```bash
go install github.com/yourusername/bqui/cmd/bqui@latest
```

### Build Locally

```bash
git clone https://github.com/yourusername/bqui.git
cd bqui
make build
```

## 🔧 Setup

### Authentication

bqui uses Google Cloud's standard authentication methods:

1. **Application Default Credentials**:
   ```bash
   gcloud auth application-default login
   ```

2. **Service Account Key**:
   ```bash
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
   ```

3. **Via Command Line**:
   ```bash
   bqui -credentials /path/to/service-account.json
   ```

### Project Configuration

Set your default project:
```bash
export GOOGLE_CLOUD_PROJECT=your-project-id
# or
export GCP_PROJECT=your-project-id
```

Or specify via command line:
```bash
bqui -project your-project-id
```

## 🎮 Usage

### Basic Usage

```bash
# Use default project from credentials
bqui

# Specify project explicitly
bqui -project my-gcp-project

# Use custom credentials
bqui -project my-project -credentials /path/to/creds.json

# Show help
bqui -h
```

### Key Bindings

#### Navigation
- `↑↓←→` or `hjkl` - Navigate lists and panes
- `Enter` - Select dataset or table
- `Tab` - Switch between left/right panes or tabs within right pane
- `g` / `G` - Go to top/bottom (vim-style)
- `Home` / `End` - Go to top/bottom
- `Page Up` / `Page Down` - Page navigation

#### Search & Filter
- `/` - Start search/filter mode
- `Esc` - Clear filter or exit search mode
- Type to filter results in real-time

#### Actions
- `y` or `Ctrl+Y` - Copy full table name to clipboard
- `Tab` - Cycle through right pane tabs (Schema → Preview → Query → Schema...)
- `Esc` - Go back to left pane / cancel search / exit help
- `Ctrl+P` - Open project selector
- `?` - Show/hide help
- `q` or `Ctrl+C` - Quit application

#### Right Pane Tabs
- **Schema Tab**: View table structure, field types, and descriptions
- **Preview Tab**: See sample data from the table
- **Query Tab**: Execute custom SQL queries (coming soon)

### Navigation Flow

1. **Start**: View all datasets in your project
2. **Select Dataset**: Press `Enter` on a dataset to see its tables
3. **Select Table**: Press `Enter` on a table to automatically switch to schema view
4. **Explore Table**: Use `Tab` to cycle through Schema → Preview → Query tabs
5. **Go Back**: Press `Esc` to return to the left pane (table list)
6. **Search**: Use `/` to quickly find datasets or tables
7. **Copy**: Use `y` to copy table names for use in your queries

## 🧪 Examples

### Exploring a Dataset

```bash
# Launch bqui
bqui -project my-data-warehouse

# Navigate to your analytics dataset
# Use ↓ or j to select "analytics_dataset"
# Press Enter to explore tables

# Search for specific tables
# Press / and type "user" to filter to user-related tables
# Press Enter on "user_events" to see its schema

# Copy table name for use in queries
# Press y to copy "my-data-warehouse.analytics_dataset.user_events"
```

## 🔧 Configuration

### Environment Variables

- `GOOGLE_APPLICATION_CREDENTIALS` - Path to service account credentials
- `GOOGLE_CLOUD_PROJECT` - Default GCP project ID
- `GCP_PROJECT` - Alternative project ID variable

### Command Line Flags

- `-project` - BigQuery project ID
- `-credentials` - Path to credentials file
- `-emulator` - BigQuery emulator endpoint (for testing)
- `-version` - Show version information

## 🧑‍💻 Development

### Prerequisites

- Go 1.21 or later
- Google Cloud SDK (for authentication)

### Building

```bash
# Install dependencies
go mod download

# Build the binary
go build ./cmd/bqui

# Run tests
make test

# Run with emulator for testing
make test-emulator
```

### Testing

bqui includes comprehensive tests using the [BigQuery emulator](https://github.com/goccy/bigquery-emulator):

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
go test ./internal/bigquery -v
```

### Project Structure

```
bqui/
├── cmd/bqui/           # Main application entry point
├── internal/
│   ├── bigquery/       # BigQuery client wrapper
│   └── tui/           # TUI components (Bubble Tea)
├── pkg/clipboard/      # Clipboard utilities
├── test/              # Test files and fixtures
└── .github/workflows/ # CI/CD pipelines
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Quick Start

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and add tests
4. Run tests: `make test`
5. Commit: `git commit -m 'Add amazing feature'`
6. Push: `git push origin feature/amazing-feature`
7. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - The fantastic TUI framework
- [BigQuery Go Client](https://pkg.go.dev/cloud.google.com/go/bigquery) - Official Google BigQuery client
- [BigQuery Emulator](https://github.com/goccy/bigquery-emulator) - Testing infrastructure
- Inspired by [lazydocker](https://github.com/jesseduffield/lazydocker) and similar TUI tools

## 🐛 Troubleshooting

### Common Issues

**Authentication Errors**
```bash
# Ensure you're authenticated
gcloud auth application-default login

# Or set credentials explicitly
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/creds.json
```

**Project Access Issues**
```bash
# Verify project access
gcloud projects list

# Set default project
gcloud config set project YOUR_PROJECT_ID
```

**Build Issues**
```bash
# Clean and rebuild
go clean -cache
go mod download
go build ./cmd/bqui
```

### Getting Help

- 📝 [Open an issue](https://github.com/yourusername/bqui/issues) for bug reports
- 💡 [Start a discussion](https://github.com/yourusername/bqui/discussions) for questions
- 📧 Email: your-email@example.com

---

*bqui makes BigQuery exploration delightful. Happy querying! 🚀*

# bqui Architecture

A clean, modular BigQuery Terminal UI built with Go and Bubble Tea.

## 📁 Project Structure

```
bqui/
├── cmd/bqui/           # Application entry point
│   └── main.go         # CLI setup, auth, project detection
├── internal/
│   ├── bigquery/       # BigQuery client wrapper
│   │   ├── client.go   # BQ operations, project switching
│   │   └── types.go    # Data structures (Dataset, Table, Column)
│   └── tui/            # Terminal UI components (Bubble Tea)
│       ├── app.go      # Main application model & key handling
│       ├── dataset_list.go    # Left pane (datasets/tables)
│       ├── table_detail.go    # Right pane (schema/preview/query)
│       ├── project_selector.go # Project switching UI
│       ├── search.go   # Search/filter input handling
│       ├── messages.go # Bubble Tea commands & messages
│       └── styles.go   # UI styling with Lip Gloss
├── pkg/clipboard/      # Cross-platform clipboard utilities
└── test/              # Tests with BigQuery emulator
```

## 🏗️ Architecture Patterns

### **1. Clean Architecture**
- **Domain**: `internal/bigquery/` - Pure business logic
- **UI**: `internal/tui/` - Terminal interface components  
- **Infrastructure**: `pkg/` - External utilities
- **App**: `cmd/` - Application bootstrap

### **2. Bubble Tea Model**
- **Model**: Application state (focus, data, UI state)
- **Update**: Event handling (keys, messages, commands)
- **View**: Rendering UI components
- **Commands**: Async operations (API calls, file ops)

## 🔄 Data Flow

```
User Input → App Model → Component Updates → BigQuery API → UI Refresh
     ↓            ↓            ↓              ↓           ↑
   KeyMsg → Update() → Commands → client.go → Messages
```

### **Key Components:**

1. **App Model** (`app.go`)
   - Central state management
   - Focus control (left/right panes)
   - Global key bindings
   - Component coordination

2. **Dataset List** (`dataset_list.go`)
   - Hierarchical navigation (datasets → tables)
   - Selection vs hovering logic
   - Search/filter functionality

3. **Table Detail** (`table_detail.go`)
   - Tab management (Schema/Preview/Query)
   - Column search in schema
   - Horizontal scrolling
   - Tabular schema rendering

4. **BigQuery Client** (`client.go`)
   - API wrapper with authentication
   - Project switching capabilities
   - Data fetching (datasets, tables, schema, preview)

## 🎛️ State Management

### **Focus States**
- `FocusDatasetList` - Left pane active
- `FocusTableDetail` - Right pane active  
- `FocusProjectSelector` - Project selection modal
- `FocusSearch` - Search input mode

### **Navigation Logic**
- **Hover**: Updates preview but maintains focus
- **Selection**: Changes focus and loads data
- **Explicit flags**: `tableSelected` distinguishes hover vs select

## 🔧 Key Design Decisions

### **1. Separation of Concerns**
- BigQuery logic isolated from UI
- Components communicate via messages
- Pure functions for rendering

### **2. Responsive UI**
- Horizontal scrolling for wide content
- Adaptive status bar layout
- Multi-line help display

### **3. Authentication Flow**
- Auto-detect gcloud default project
- Support multiple credential methods
- Graceful fallbacks

### **4. Performance**
- Lazy loading of table data
- Filtered rendering for large datasets
- Efficient state updates

## 🎨 UI Architecture

### **Layout System** (Lip Gloss)
```
┌─────────────┬─────────────────────────┐
│ Datasets    │ Schema | Preview | Query│
│ ├─dataset1  │ ┌─────────────────────┐ │
│ ├─dataset2  │ │ Field  Type  Mode   │ │
│ └─table1◄   │ │ id     INT64 NULLABLE│ │
│             │ └─────────────────────┘ │
└─────────────┴─────────────────────────┘
Search: [filter text]
Status: Loaded schema | Press ? for help
```

### **Component Hierarchy**
```
App (root)
├── DatasetList (left pane)
├── TableDetail (right pane)  
│   ├── Schema Tab
│   ├── Preview Tab
│   └── Query Tab
├── ProjectSelector (modal)
└── Search (input overlay)
```

## 🔄 Message Patterns

### **Async Operations**
```go
type DatasetsLoadedMsg struct {
    Datasets []*bigquery.Dataset
}

func (m Model) loadDatasets() tea.Cmd {
    return func() tea.Msg {
        datasets, err := m.bqClient.ListDatasets()
        // ... handle error
        return DatasetsLoadedMsg{Datasets: datasets}
    }
}
```

### **User Interactions**
```go
case key.Matches(msg, m.keyMap.Enter):
    // Navigation logic
case key.Matches(msg, m.keyMap.Search):
    // Search mode
case key.Matches(msg, m.keyMap.Copy):
    // Clipboard operations
```

## 🧪 Testing Strategy

- **Unit Tests**: Core business logic
- **Integration Tests**: BigQuery emulator
- **UI Tests**: Component behavior
- **E2E Tests**: Full user workflows

## 📦 Build System

- **Makefile**: Unified build commands
- **GitHub Actions**: CI/CD with Make integration
- **Multi-platform**: Cross-compilation support
- **Go Modules**: Dependency management

---

**Design Philosophy**: Simple, fast, keyboard-driven BigQuery exploration that feels native to terminal users.
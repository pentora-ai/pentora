package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDAGFromFile_YAML(t *testing.T) {
	// Create temporary YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "test.yaml")

	yamlContent := `name: "Test DAG"
version: "1.0"
description: "Test DAG for loading"

nodes:
  - id: "node-1"
    module: "test.module"
    produces:
      - "data.1"

  - id: "node-2"
    module: "test.module"
    depends_on:
      - "node-1"
    consumes:
      - "data.1"
    produces:
      - "data.2"
`

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	// Load DAG
	dag, err := LoadDAGFromFile(yamlPath, false)
	require.NoError(t, err)
	require.NotNil(t, dag)

	// Verify structure
	require.Equal(t, "Test DAG", dag.Name)
	require.Equal(t, "1.0", dag.Version)
	require.Len(t, dag.Nodes, 2)
	require.Equal(t, "node-1", dag.Nodes[0].ID)
	require.Equal(t, "node-2", dag.Nodes[1].ID)
	require.Equal(t, []string{"node-1"}, dag.Nodes[1].DependsOn)
}

func TestLoadDAGFromFile_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "test.json")

	jsonContent := `{
  "name": "Test DAG",
  "version": "1.0",
  "nodes": [
    {
      "id": "node-1",
      "module": "test.module",
      "produces": ["data.1"]
    },
    {
      "id": "node-2",
      "module": "test.module",
      "depends_on": ["node-1"],
      "consumes": ["data.1"],
      "produces": ["data.2"]
    }
  ]
}`

	err := os.WriteFile(jsonPath, []byte(jsonContent), 0o644)
	require.NoError(t, err)

	// Load DAG
	dag, err := LoadDAGFromFile(jsonPath, false)
	require.NoError(t, err)
	require.NotNil(t, dag)

	require.Equal(t, "Test DAG", dag.Name)
	require.Len(t, dag.Nodes, 2)
}

func TestLoadDAGFromFile_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(txtPath, []byte("invalid"), 0o644)
	require.NoError(t, err)

	_, err = LoadDAGFromFile(txtPath, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported file format")
}

func TestLoadDAGFromFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "invalid.yaml")

	// Invalid YAML (unmatched bracket)
	err := os.WriteFile(yamlPath, []byte("name: [invalid"), 0o644)
	require.NoError(t, err)

	_, err = LoadDAGFromFile(yamlPath, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse YAML")
}

func TestLoadDAGFromFile_ValidationError(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "invalid-dag.yaml")

	// DAG with cycle
	yamlContent := `name: "Invalid DAG"
nodes:
  - id: "node-a"
    module: "test.module"
    depends_on:
      - "node-b"
    produces:
      - "data.a"

  - id: "node-b"
    module: "test.module"
    depends_on:
      - "node-a"
    produces:
      - "data.b"
`

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	_, err = LoadDAGFromFile(yamlPath, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "validation failed")
	require.Contains(t, err.Error(), "cycle")
}

func TestLoadDAGFromFile_SkipValidation(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "invalid-dag.yaml")

	// DAG with cycle
	yamlContent := `name: "Invalid DAG"
nodes:
  - id: "node-a"
    module: "test.module"
    depends_on:
      - "node-b"
    produces:
      - "data.a"

  - id: "node-b"
    module: "test.module"
    depends_on:
      - "node-a"
    produces:
      - "data.b"
`

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	// Should succeed when validation is skipped
	dag, err := LoadDAGFromFile(yamlPath, true)
	require.NoError(t, err)
	require.NotNil(t, dag)
}

func TestLoadDAGFromFile_FileNotFound(t *testing.T) {
	_, err := LoadDAGFromFile("/nonexistent/file.yaml", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "read file")
}

func TestLoadDAGFromBytes_YAML(t *testing.T) {
	yamlContent := []byte(`name: "Test DAG"
nodes:
  - id: "node-1"
    module: "test.module"
    produces:
      - "data.1"
`)

	dag, err := LoadDAGFromBytes(yamlContent, false)
	require.NoError(t, err)
	require.NotNil(t, dag)
	require.Equal(t, "Test DAG", dag.Name)
	require.Len(t, dag.Nodes, 1)
}

func TestLoadDAGFromBytes_JSON(t *testing.T) {
	jsonContent := []byte(`{
  "name": "Test DAG",
  "nodes": [
    {
      "id": "node-1",
      "module": "test.module",
      "produces": ["data.1"]
    }
  ]
}`)

	dag, err := LoadDAGFromBytes(jsonContent, false)
	require.NoError(t, err)
	require.NotNil(t, dag)
	require.Equal(t, "Test DAG", dag.Name)
}

func TestLoadDAGFromBytes_Invalid(t *testing.T) {
	_, err := LoadDAGFromBytes([]byte("invalid content"), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse YAML/JSON")
}

func TestSaveDAGToFile_YAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "output.yaml")

	dag := &DAGSchema{
		Name:    "Test DAG",
		Version: "1.0",
		Nodes: []DAGNode{
			{
				ID:       "node-1",
				Module:   "test.module",
				Produces: []string{"data.1"},
			},
		},
	}

	err := SaveDAGToFile(dag, yamlPath)
	require.NoError(t, err)

	// Verify file exists and is valid
	loadedDAG, err := LoadDAGFromFile(yamlPath, false)
	require.NoError(t, err)
	require.Equal(t, dag.Name, loadedDAG.Name)
	require.Equal(t, dag.Version, loadedDAG.Version)
	require.Len(t, loadedDAG.Nodes, 1)
}

func TestSaveDAGToFile_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "output.json")

	dag := &DAGSchema{
		Name:    "Test DAG",
		Version: "1.0",
		Nodes: []DAGNode{
			{
				ID:       "node-1",
				Module:   "test.module",
				Produces: []string{"data.1"},
			},
		},
	}

	err := SaveDAGToFile(dag, jsonPath)
	require.NoError(t, err)

	// Verify file exists and is valid
	loadedDAG, err := LoadDAGFromFile(jsonPath, false)
	require.NoError(t, err)
	require.Equal(t, dag.Name, loadedDAG.Name)
}

func TestSaveDAGToFile_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "output.txt")

	dag := &DAGSchema{
		Name:  "Test DAG",
		Nodes: []DAGNode{},
	}

	err := SaveDAGToFile(dag, txtPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported file format")
}

func TestSaveDAGToFile_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "subdir", "nested", "output.yaml")

	dag := &DAGSchema{
		Name:  "Test DAG",
		Nodes: []DAGNode{{ID: "node-1", Module: "test.module"}},
	}

	err := SaveDAGToFile(dag, nestedPath)
	require.NoError(t, err)

	// Verify directory was created
	_, err = os.Stat(filepath.Dir(nestedPath))
	require.NoError(t, err)
}

func TestDAGFromModules_Simple(t *testing.T) {
	// Create mock modules
	modules := []Module{
		&dagMockModule{
			meta: ModuleMetadata{
				Name: "discovery.icmp-ping",
				Produces: []DataContractEntry{
					{Key: "discovery.live_hosts"},
				},
			},
		},
		&dagMockModule{
			meta: ModuleMetadata{
				Name: "scan.tcp-port-scanner",
				Consumes: []DataContractEntry{
					{Key: "discovery.live_hosts"},
				},
				Produces: []DataContractEntry{
					{Key: "scan.open_ports"},
				},
			},
		},
	}

	dag, err := DAGFromModules("Test Workflow", modules)
	require.NoError(t, err)
	require.NotNil(t, dag)

	require.Equal(t, "Test Workflow", dag.Name)
	require.Equal(t, "1.0", dag.Version)
	require.Len(t, dag.Nodes, 2)

	// Verify node IDs
	require.Equal(t, "icmp-ping-0", dag.Nodes[0].ID)
	require.Equal(t, "tcp-port-1", dag.Nodes[1].ID)

	// Verify dependencies inferred from data flow
	require.Equal(t, []string{"icmp-ping-0"}, dag.Nodes[1].DependsOn)
}

func TestGenerateNodeID(t *testing.T) {
	tests := []struct {
		name       string
		moduleName string
		index      int
		want       string
	}{
		{
			name:       "Simple module",
			moduleName: "discovery.icmp-ping",
			index:      0,
			want:       "icmp-ping-0",
		},
		{
			name:       "Scanner suffix",
			moduleName: "scan.tcp-port-scanner",
			index:      1,
			want:       "tcp-port-1",
		},
		{
			name:       "Parser suffix",
			moduleName: "parse.ssh-parser",
			index:      2,
			want:       "ssh-2",
		},
		{
			name:       "No namespace",
			moduleName: "simple",
			index:      3,
			want:       "node-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateNodeID(tt.moduleName, tt.index)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestExtractDataKeys(t *testing.T) {
	contracts := []DataContractEntry{
		{Key: "discovery.live_hosts", Description: "Live hosts"},
		{Key: "scan.open_ports", Description: "Open ports"},
	}

	keys := extractDataKeys(contracts)
	require.Equal(t, []string{"discovery.live_hosts", "scan.open_ports"}, keys)
}

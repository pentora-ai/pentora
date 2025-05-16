// pkg/scan/dag.go
package scan

import (
	"errors"
	"fmt"
	"slices"
)

// dagNode represents a plugin and its dependencies.
type dagNode struct {
	Name     string
	Deps     []string
	Visited  bool
	TempMark bool
}

// buildGraph creates a map of plugin nodes keyed by name.
func buildGraph(plugins []namedPlugin) map[string]*dagNode {
	graph := make(map[string]*dagNode)
	for _, p := range plugins {
		graph[p.Name] = &dagNode{
			Name: p.Name,
			Deps: p.DependsOn,
		}
	}
	return graph
}

// topologicalSort sorts plugins by dependency order using DFS.
func topologicalSort(graph map[string]*dagNode) ([]string, error) {
	var result []string
	var visit func(n *dagNode) error

	visit = func(n *dagNode) error {
		if n.TempMark {
			return fmt.Errorf("cyclic dependency detected at %s", n.Name)
		}
		if n.Visited {
			return nil
		}
		n.TempMark = true
		for _, dep := range n.Deps {
			d, ok := graph[dep]
			if !ok {
				return fmt.Errorf("missing dependency '%s' for plugin '%s'", dep, n.Name)
			}
			if err := visit(d); err != nil {
				return err
			}
		}
		n.TempMark = false
		n.Visited = true
		result = append(result, n.Name)
		return nil
	}

	for _, node := range graph {
		if !node.Visited {
			if err := visit(node); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// namedPlugin is a simplified struct used for sorting.
type namedPlugin struct {
	Name      string
	DependsOn []string
}

// dagLayers returns plugins organized into execution layers (parallel per layer).
func dagLayers(graph map[string]*dagNode) ([][]string, error) {
	inDegree := make(map[string]int)
	for _, node := range graph {
		for range node.Deps {
			inDegree[node.Name]++
		}
	}

	// First layer: those without dependencies
	var layers [][]string
	var current []string
	for name := range graph {
		if inDegree[name] == 0 {
			current = append(current, name)
		}
	}
	if len(current) == 0 {
		return nil, errors.New("no independent plugins found (cyclic dependency?)")
	}

	// Gradually build the layers
	visited := make(map[string]bool)
	for len(current) > 0 {
		layers = append(layers, current)
		next := []string{}
		for _, name := range current {
			visited[name] = true
			for _, node := range graph {
				if contains(node.Deps, name) {
					inDegree[node.Name]--
					if inDegree[node.Name] == 0 && !visited[node.Name] {
						next = append(next, node.Name)
					}
				}
			}
		}
		current = next
	}

	// Final check: have all nodes been processed?
	if len(visited) != len(graph) {
		return nil, errors.New("cyclic dependency detected")
	}

	return layers, nil
}

// contains checks if a string slice contains a value.
func contains(slice []string, val string) bool {
	return slices.Contains(slice, val)
}

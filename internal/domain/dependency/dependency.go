package dependency

import "fmt"

type Edge struct {
	JobID          string
	DependsOnJobID string
}

func ValidateGraph(edges []Edge) error {
	graph := make(map[string][]string)
	for _, edge := range edges {
		if edge.JobID == edge.DependsOnJobID {
			return fmt.Errorf("job %s cannot depend on itself", edge.JobID)
		}
		graph[edge.JobID] = append(graph[edge.JobID], edge.DependsOnJobID)
	}

	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	for node := range graph {
		if hasCycle(node, graph, visiting, visited) {
			return fmt.Errorf("dependency graph contains a cycle")
		}
	}
	return nil
}

func hasCycle(node string, graph map[string][]string, visiting, visited map[string]bool) bool {
	if visited[node] {
		return false
	}
	if visiting[node] {
		return true
	}

	visiting[node] = true
	for _, next := range graph[node] {
		if hasCycle(next, graph, visiting, visited) {
			return true
		}
	}
	visiting[node] = false
	visited[node] = true
	return false
}

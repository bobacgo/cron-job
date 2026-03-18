package dependency

import "testing"

func TestValidateGraphCycle(t *testing.T) {
	edges := []Edge{
		{JobID: "a", DependsOnJobID: "b"},
		{JobID: "b", DependsOnJobID: "c"},
		{JobID: "c", DependsOnJobID: "a"},
	}
	if err := ValidateGraph(edges); err == nil {
		t.Fatalf("ValidateGraph() error = nil, want cycle error")
	}
}

func TestValidateGraphAcyclic(t *testing.T) {
	edges := []Edge{
		{JobID: "b", DependsOnJobID: "a"},
		{JobID: "c", DependsOnJobID: "b"},
	}
	if err := ValidateGraph(edges); err != nil {
		t.Fatalf("ValidateGraph() error = %v, want nil", err)
	}
}

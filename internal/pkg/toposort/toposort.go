// Package toposort 提供有向无环图的拓扑排序。
//
// 本实现源自 github.com/gammazero/toposort v0.1.1，按 MIT 许可证分发，详见本目录的 LICENSE 文件。
package toposort

import (
	"errors"
	"fmt"
)

// Edge represents a pair of vertexes.  Each vertex is an opaque type.
type Edge [2]interface{}

// Toposort performs a topological sort of the DAG defined by given edges.
//
// Takes a slice of Edge, where each element is a vertex pair representing an
// edge in the graph.  Each pair can also be considered a dependency
// relationship where Edge[0] must happen before Edge[1].  For a reversed
// order, call ToposortR().
//
// To include a node that is not connected to the rest of the graph, include a
// node with one nil vertex.  It can appear anywhere in the sorted output.
//
// Returns an ordered list of vertexes where each vertex occurs before any of
// its destination vertexes.  An error is returned if a cycle is detected.
func Toposort(edges []Edge) ([]interface{}, error) {
	g, err := makeGraph(edges)
	if err != nil {
		return nil, err
	}
	sorted := make([]interface{}, 0, len(g))

	// Create map of vertexes to incoming edge count, and set counts to 0
	inDegree := make(map[interface{}]int, len(g))
	for n := range g {
		inDegree[n] = 0
	}

	// For each vertex u, get adjacent list
	for _, adjacent := range g {
		// For each vertex v adjacent to u
		for _, v := range adjacent {
			// Increment inDegree[v]
			inDegree[v]++
		}
	}

	// Make a list next consisting of all vertexes u such that inDegree[u] = 0
	var next []interface{}
	for u, deg := range inDegree {
		if deg == 0 {
			next = append(next, u)
		}
	}

	// While next is not empty...
	for len(next) > 0 {
		// Pop a vertex from next and call it vertex u
		u := next[len(next)-1]
		next = next[:len(next)-1]

		// Add u to the end sorted list
		sorted = append(sorted, u)

		// For each vertex v adjacent to sorted vertex u
		for _, v := range g[u] {
			// Decrement count of incoming edges
			inDegree[v]--
			// Enqueue nodes with no incoming edges
			if inDegree[v] == 0 {
				next = append(next, v)
			}
		}
	}

	// Check for cycle
	if len(sorted) < len(g) {
		var cycleNodes []string
		for u, deg := range inDegree {
			if deg != 0 {
				cycleNodes = append(cycleNodes, fmt.Sprint(u))
			}
		}
		return nil, fmt.Errorf("graph contains cycle in nodes %s", cycleNodes)
	}

	// Return the sorted vertex list
	return sorted, nil
}

// ToposortR is the same as Toposort with the order of the output reversed.
// This has the same effect as changing the vertex order of each edge.
func ToposortR(edges []Edge) ([]interface{}, error) {
	sorted, err := Toposort(edges)
	if err != nil {
		return nil, err
	}
	// Reverse slice
	for i := len(sorted)/2 - 1; i >= 0; i-- {
		opp := len(sorted) - 1 - i
		sorted[i], sorted[opp] = sorted[opp], sorted[i]
	}
	return sorted, nil
}

// makeGraph creates a map of source node to destination nodes.  An edge with
// only one vertex is added to the graph, if it is not already in the graph.
func makeGraph(edges []Edge) (map[interface{}][]interface{}, error) {
	graph := make(map[interface{}][]interface{}, len(edges)+1)
	for i := range edges {
		u, v := edges[i][0], edges[i][1]
		if u == v {
			return nil, errors.New("nodes in edge cannot be the same")
		}
		if u == nil {
			// Add vertex only (empty destination list)
			if _, ok := graph[v]; !ok {
				graph[v] = nil
			}
		} else if v == nil {
			if _, ok := graph[u]; !ok {
				graph[u] = nil
			}
		} else {
			graph[u] = append(graph[u], v)
		}
	}
	return graph, nil
}

package tree

import (
	"os"

	"github.com/benjaminchristie/go-arxiv-tree/internal/api"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

func contains(v *[]*Node, c *Node) bool {
	for _, e := range *v {
		if c == e {
			return true
		}
	}
	return false
}

func traversal(n *Node, visitedNodes *[]*Node, pstr string, cb func(*Node, string) string) {
	for _, child := range n.Cites {
		cstr := cb(child, pstr)
		if contains(visitedNodes, child) {
			continue
		}
		*visitedNodes = append(*visitedNodes, child)
		traversal(child, visitedNodes, cstr, cb)
	}
}

func Visualize(n *Node, filename string) error {
	g := graph.New(graph.StringHash, graph.Directed())
	cb := func(c *Node, pstr string) string {
		_, t, err := api.QueryBibtexEntry(c.Entry)
		if err != nil {
			t = "Error"
		}
		g.AddVertex(t)
		g.AddEdge(pstr, t)
		return t
	}
	_, t, err := api.QueryBibtexEntry(n.Entry)
	if err != nil {
		t = "Error"
	}
	visitedNodes := make([]*Node, 0)
	g.AddVertex(t)
	traversal(n, &visitedNodes, t, cb)
	file, _ := os.Create(filename)
	err = draw.DOT(g, file)
	return err
}

// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package draw

// hv_line_graph.go helps to keep track of locations where lines cross.

import (
	"fmt"
	"image"

	"github.com/mum4k/termdash/linestyle"
)

// hVLineEdge is an edge between two points on the graph.
type hVLineEdge struct {
	// from is the starting node of this edge.
	// From is guaranteed to be less than to.
	from image.Point

	// to is the ending point of this edge.
	to image.Point
}

// newHVLineEdge returns a new edge between the two points.
func newHVLineEdge(from, to image.Point) hVLineEdge {
	return hVLineEdge{
		from: from,
		to:   to,
	}
}

// hVLineNode represents one node in the graph.
// I.e. one cell.
type hVLineNode struct {
	// p is the point where this node is.
	p image.Point

	// edges are the edges between this node and the surrounding nodes.
	// The code only supports horizontal and vertical lines so there can only
	// ever be edges to nodes on these planes.
	edges map[hVLineEdge]bool
}

// newHVLineNode creates a new newHVLineNode.
func newHVLineNode(p image.Point) *hVLineNode {
	return &hVLineNode{
		p:     p,
		edges: map[hVLineEdge]bool{},
	}
}

// hasDown determines if this node has an edge to the one below it.
func (n *hVLineNode) hasDown() bool {
	target := newHVLineEdge(n.p, image.Point{n.p.X, n.p.Y + 1})
	_, ok := n.edges[target]
	return ok
}

// hasUp determines if this node has an edge to the one above it.
func (n *hVLineNode) hasUp() bool {
	target := newHVLineEdge(image.Point{n.p.X, n.p.Y - 1}, n.p)
	_, ok := n.edges[target]
	return ok
}

// hasLeft determines if this node has an edge to the next node on the left.
func (n *hVLineNode) hasLeft() bool {
	target := newHVLineEdge(image.Point{n.p.X - 1, n.p.Y}, n.p)
	_, ok := n.edges[target]
	return ok
}

// hasRight determines if this node has an edge to the next node on the right.
func (n *hVLineNode) hasRight() bool {
	target := newHVLineEdge(n.p, image.Point{n.p.X + 1, n.p.Y})
	_, ok := n.edges[target]
	return ok
}

// rune, given the selected line style returns the correct line character to
// represent this node.
// Only handles nodes with two or more edges, as returned by multiEdgeNodes().
func (n *hVLineNode) rune(ls linestyle.LineStyle) (rune, error) {
	parts, err := lineParts(ls)
	if err != nil {
		return -1, err
	}

	switch len(n.edges) {
	case 2:
		switch {
		case n.hasLeft() && n.hasRight():
			return parts[hLine], nil
		case n.hasUp() && n.hasDown():
			return parts[vLine], nil
		case n.hasDown() && n.hasRight():
			return parts[topLeftCorner], nil
		case n.hasDown() && n.hasLeft():
			return parts[topRightCorner], nil
		case n.hasUp() && n.hasRight():
			return parts[bottomLeftCorner], nil
		case n.hasUp() && n.hasLeft():
			return parts[bottomRightCorner], nil
		default:
			return -1, fmt.Errorf("unexpected two edges in node representing point %v: %v", n.p, n.edges)
		}

	case 3:
		switch {
		case n.hasUp() && n.hasLeft() && n.hasRight():
			return parts[hAndUp], nil
		case n.hasDown() && n.hasLeft() && n.hasRight():
			return parts[hAndDown], nil
		case n.hasUp() && n.hasDown() && n.hasRight():
			return parts[vAndRight], nil
		case n.hasUp() && n.hasDown() && n.hasLeft():
			return parts[vAndLeft], nil

		default:
			return -1, fmt.Errorf("unexpected three edges in node representing point %v: %v", n.p, n.edges)
		}

	case 4:
		return parts[vAndH], nil
	default:
		return -1, fmt.Errorf("unexpected number of edges(%d) in node representing point %v", len(n.edges), n.p)
	}
}

// hVLineGraph represents lines on the canvas as a bidirectional graph of
// nodes. Helps to determine the characters that should be used where multiple
// lines cross.
type hVLineGraph struct {
	nodes map[image.Point]*hVLineNode
}

// newHVLineGraph creates a new hVLineGraph.
func newHVLineGraph() *hVLineGraph {
	return &hVLineGraph{
		nodes: make(map[image.Point]*hVLineNode),
	}
}

// getOrCreateNode gets an existing or creates a new node for the point.
func (g *hVLineGraph) getOrCreateNode(p image.Point) *hVLineNode {
	if n, ok := g.nodes[p]; ok {
		return n
	}
	n := newHVLineNode(p)
	g.nodes[p] = n
	return n
}

// addLine adds a line to the graph.
// This adds edges between all the points on the line.
func (g *hVLineGraph) addLine(line *hVLine) {
	switch {
	case line.horizontal():
		for curX := line.start.X; curX < line.end.X; curX++ {
			from := image.Point{curX, line.start.Y}
			to := image.Point{curX + 1, line.start.Y}
			n1 := g.getOrCreateNode(from)
			n2 := g.getOrCreateNode(to)
			edge := newHVLineEdge(from, to)
			n1.edges[edge] = true
			n2.edges[edge] = true
		}

	case line.vertical():
		for curY := line.start.Y; curY < line.end.Y; curY++ {
			from := image.Point{line.start.X, curY}
			to := image.Point{line.start.X, curY + 1}
			n1 := g.getOrCreateNode(from)
			n2 := g.getOrCreateNode(to)
			edge := newHVLineEdge(from, to)
			n1.edges[edge] = true
			n2.edges[edge] = true
		}
	}
}

// multiEdgeNodes returns all nodes that have more than one edge.  These are
// the nodes where we might need to use different line characters to represent
// the crossing of multiple lines.
func (g *hVLineGraph) multiEdgeNodes() []*hVLineNode {
	var nodes []*hVLineNode
	for _, n := range g.nodes {
		if len(n.edges) <= 1 {
			continue
		}
		nodes = append(nodes, n)
	}
	return nodes
}

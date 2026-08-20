package document

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
)

// NodeID identifies a node for the lifetime of a document.
type NodeID uint64

// NoNode is the absence of a node.
const NoNode NodeID = 0

// Node is one entry in an ordered tree.
type Node struct {
	ID       NodeID
	Text     string
	Parent   NodeID
	Children []NodeID
}

// Branch is a detached subtree used by copy and paste operations.
type Branch struct {
	Text     string
	Children []Branch
}

// Document stores an ordered forest. Most maps contain one root, but accepting
// a forest lets Beech open indentation-based files without changing them.
type Document struct {
	nodes  map[NodeID]Node
	roots  []NodeID
	nextID NodeID
}

// New creates a document with one root node.
func New(rootText string) *Document {
	d := NewEmpty()
	d.AppendRoot(rootText)
	return d
}

// NewEmpty creates a document without roots. It is intended for parsers.
func NewEmpty() *Document {
	return &Document{
		nodes:  make(map[NodeID]Node),
		nextID: 1,
	}
}

// Clone returns a deep copy of the document.
func (d *Document) Clone() *Document {
	if d == nil {
		return nil
	}
	clone := &Document{
		nodes:  make(map[NodeID]Node, len(d.nodes)),
		roots:  slices.Clone(d.roots),
		nextID: d.nextID,
	}
	for id, node := range d.nodes {
		node.Children = slices.Clone(node.Children)
		clone.nodes[id] = node
	}
	return clone
}

// Len returns the number of nodes.
func (d *Document) Len() int {
	if d == nil {
		return 0
	}
	return len(d.nodes)
}

// Roots returns the top-level nodes in document order.
func (d *Document) Roots() []NodeID {
	if d == nil {
		return nil
	}
	return slices.Clone(d.roots)
}

// Node returns a copy of one node.
func (d *Document) Node(id NodeID) (Node, bool) {
	if d == nil {
		return Node{}, false
	}
	node, ok := d.nodes[id]
	if !ok {
		return Node{}, false
	}
	node.Children = slices.Clone(node.Children)
	return node, true
}

// Text returns a node's text, or the empty string for an unknown node.
func (d *Document) Text(id NodeID) string {
	if d == nil {
		return ""
	}
	return d.nodes[id].Text
}

// Parent returns a node's parent, or NoNode for a root or unknown node.
func (d *Document) Parent(id NodeID) NodeID {
	if d == nil {
		return NoNode
	}
	return d.nodes[id].Parent
}

// Children returns a node's children in document order.
func (d *Document) Children(id NodeID) []NodeID {
	node, _ := d.Node(id)
	return node.Children
}

// AppendRoot adds a root at the end of the forest.
func (d *Document) AppendRoot(text string) NodeID {
	return d.insertNode(NoNode, len(d.roots), text)
}

// AppendChild adds a child to parent.
func (d *Document) AppendChild(parent NodeID, text string) (NodeID, error) {
	node, ok := d.nodes[parent]
	if !ok {
		return NoNode, fmt.Errorf("parent %d does not exist", parent)
	}
	return d.insertNode(parent, len(node.Children), text), nil
}

// AddChild adds a child after the parent's existing children.
func (d *Document) AddChild(parent NodeID, text string) (NodeID, bool) {
	id, err := d.AppendChild(parent, text)
	return id, err == nil
}

// AddSiblingAfter adds a node immediately after id.
func (d *Document) AddSiblingAfter(id NodeID, text string) (NodeID, bool) {
	node, ok := d.nodes[id]
	if !ok {
		return NoNode, false
	}
	siblings := d.siblings(node.Parent)
	index := slices.Index(siblings, id)
	if index < 0 {
		return NoNode, false
	}
	return d.insertNode(node.Parent, index+1, text), true
}

// Rename changes a node's text. Empty text is rejected.
func (d *Document) Rename(id NodeID, text string) bool {
	node, ok := d.nodes[id]
	text = cleanText(text)
	if !ok || text == "" {
		return false
	}
	node.Text = text
	d.nodes[id] = node
	return true
}

// Delete removes a node and all descendants. The final root cannot be removed.
func (d *Document) Delete(id NodeID) bool {
	node, ok := d.nodes[id]
	if !ok || node.Parent == NoNode && len(d.roots) == 1 {
		return false
	}
	d.removeFromSiblings(node.Parent, id)
	d.deleteSubtree(id)
	return true
}

// MoveSibling moves a node by delta positions among its siblings.
func (d *Document) MoveSibling(id NodeID, delta int) bool {
	if delta == 0 {
		return false
	}
	node, ok := d.nodes[id]
	if !ok {
		return false
	}
	siblings := d.siblings(node.Parent)
	from := slices.Index(siblings, id)
	to := from + delta
	if from < 0 || to < 0 || to >= len(siblings) {
		return false
	}
	siblings[from], siblings[to] = siblings[to], siblings[from]
	d.setSiblings(node.Parent, siblings)
	return true
}

// Promote moves a non-root node after its parent.
func (d *Document) Promote(id NodeID) bool {
	node, ok := d.nodes[id]
	if !ok || node.Parent == NoNode {
		return false
	}
	parent := d.nodes[node.Parent]
	grandparent := parent.Parent
	parentSiblings := d.siblings(grandparent)
	parentIndex := slices.Index(parentSiblings, parent.ID)
	if parentIndex < 0 {
		return false
	}

	d.removeFromSiblings(parent.ID, id)
	parentSiblings = d.siblings(grandparent)
	parentIndex = slices.Index(parentSiblings, parent.ID)
	d.insertExisting(grandparent, parentIndex+1, id)
	node.Parent = grandparent
	d.nodes[id] = node
	return true
}

// Demote moves a node under its previous sibling.
func (d *Document) Demote(id NodeID) bool {
	node, ok := d.nodes[id]
	if !ok {
		return false
	}
	siblings := d.siblings(node.Parent)
	index := slices.Index(siblings, id)
	if index <= 0 {
		return false
	}
	newParent := siblings[index-1]
	d.removeFromSiblings(node.Parent, id)
	parentNode := d.nodes[newParent]
	d.insertExisting(newParent, len(parentNode.Children), id)
	node.Parent = newParent
	d.nodes[id] = node
	return true
}

// CopyBranch returns a detached copy of id and its descendants.
func (d *Document) CopyBranch(id NodeID) (Branch, bool) {
	node, ok := d.nodes[id]
	if !ok {
		return Branch{}, false
	}
	branch := Branch{Text: node.Text, Children: make([]Branch, 0, len(node.Children))}
	for _, child := range node.Children {
		copied, _ := d.CopyBranch(child)
		branch.Children = append(branch.Children, copied)
	}
	return branch, true
}

// PasteChild clones branch as the last child of parent.
func (d *Document) PasteChild(parent NodeID, branch Branch) (NodeID, bool) {
	parentNode, ok := d.nodes[parent]
	if !ok {
		return NoNode, false
	}
	return d.insertBranch(parent, len(parentNode.Children), branch), true
}

// PasteSibling clones branch immediately after id.
func (d *Document) PasteSibling(id NodeID, branch Branch) (NodeID, bool) {
	node, ok := d.nodes[id]
	if !ok {
		return NoNode, false
	}
	siblings := d.siblings(node.Parent)
	index := slices.Index(siblings, id)
	if index < 0 {
		return NoNode, false
	}
	return d.insertBranch(node.Parent, index+1, branch), true
}

// Validate checks all structural invariants.
func (d *Document) Validate() error {
	if d == nil {
		return errors.New("document is nil")
	}
	if len(d.roots) == 0 {
		return errors.New("document has no roots")
	}
	seen := make(map[NodeID]bool, len(d.nodes))
	active := make(map[NodeID]bool, len(d.nodes))
	var visit func(NodeID, NodeID) error
	visit = func(id, expectedParent NodeID) error {
		if active[id] {
			return fmt.Errorf("cycle at node %d", id)
		}
		if seen[id] {
			return fmt.Errorf("node %d appears more than once", id)
		}
		node, ok := d.nodes[id]
		if !ok {
			return fmt.Errorf("node %d is missing", id)
		}
		if node.Parent != expectedParent {
			return fmt.Errorf("node %d has parent %d, want %d", id, node.Parent, expectedParent)
		}
		if cleanText(node.Text) == "" {
			return fmt.Errorf("node %d has empty text", id)
		}
		seen[id] = true
		active[id] = true
		for _, child := range node.Children {
			if err := visit(child, id); err != nil {
				return err
			}
		}
		active[id] = false
		return nil
	}
	for _, root := range d.roots {
		if err := visit(root, NoNode); err != nil {
			return err
		}
	}
	if len(seen) != len(d.nodes) {
		return fmt.Errorf("%d unreachable nodes", len(d.nodes)-len(seen))
	}
	return nil
}

func (d *Document) insertNode(parent NodeID, index int, text string) NodeID {
	id := d.nextID
	d.nextID++
	d.nodes[id] = Node{ID: id, Text: normalizedText(text), Parent: parent}
	d.insertExisting(parent, index, id)
	return id
}

func (d *Document) insertBranch(parent NodeID, index int, branch Branch) NodeID {
	id := d.insertNode(parent, index, branch.Text)
	for _, child := range branch.Children {
		d.insertBranch(id, len(d.nodes[id].Children), child)
	}
	return id
}

func (d *Document) insertExisting(parent NodeID, index int, id NodeID) {
	siblings := d.siblings(parent)
	index = min(max(index, 0), len(siblings))
	siblings = append(siblings, NoNode)
	copy(siblings[index+1:], siblings[index:])
	siblings[index] = id
	d.setSiblings(parent, siblings)
}

func (d *Document) removeFromSiblings(parent, id NodeID) {
	siblings := d.siblings(parent)
	index := slices.Index(siblings, id)
	if index < 0 {
		return
	}
	siblings = append(siblings[:index], siblings[index+1:]...)
	d.setSiblings(parent, siblings)
}

func (d *Document) siblings(parent NodeID) []NodeID {
	if parent == NoNode {
		return slices.Clone(d.roots)
	}
	node := d.nodes[parent]
	return slices.Clone(node.Children)
}

// setSiblings stores a slice of sibling IDs. Callers must pass a slice they
// own, because it is stored without copying.
func (d *Document) setSiblings(parent NodeID, siblings []NodeID) {
	if parent == NoNode {
		d.roots = siblings
		return
	}
	node := d.nodes[parent]
	node.Children = siblings
	d.nodes[parent] = node
}

func (d *Document) deleteSubtree(id NodeID) {
	node := d.nodes[id]
	for _, child := range node.Children {
		d.deleteSubtree(child)
	}
	delete(d.nodes, id)
}

func normalizedText(text string) string {
	if text = cleanText(text); text != "" {
		return text
	}
	return "New node"
}

func cleanText(text string) string {
	text = strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', '\t':
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
	return strings.Join(strings.Fields(text), " ")
}

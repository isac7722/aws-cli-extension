package ssm

import (
	"sort"
	"strings"
	"time"
)

// NodeType distinguishes between folder (path prefix) and parameter (leaf) nodes.
type NodeType int

const (
	// NodeFolder represents an intermediate path segment (e.g. "/app/config/").
	NodeFolder NodeType = iota
	// NodeParameter represents a leaf SSM parameter.
	NodeParameter
)

// String returns a human-readable label for the node type.
func (t NodeType) String() string {
	switch t {
	case NodeFolder:
		return "folder"
	case NodeParameter:
		return "parameter"
	default:
		return "unknown"
	}
}

// ParameterMeta holds metadata for an SSM parameter leaf node.
// Fields are populated from the AWS SSM API response.
type ParameterMeta struct {
	// Type is the SSM parameter type: String, StringList, or SecureString.
	Type string
	// Version is the parameter version number.
	Version int64
	// LastModified is the timestamp of the last parameter update.
	LastModified time.Time
	// ARN is the full Amazon Resource Name.
	ARN string
	// DataType is the data type of the parameter (e.g. "text", "aws:ec2:image").
	DataType string
}

// TreeNode represents a single node in the SSM parameter path hierarchy.
// Folder nodes have children; parameter nodes carry metadata instead.
type TreeNode struct {
	// Name is the segment name (last part of the path, e.g. "db_host").
	Name string
	// Path is the full SSM parameter path (e.g. "/app/config/db_host").
	Path string
	// Type indicates whether this node is a folder or a parameter.
	Type NodeType
	// Children holds child nodes for folder nodes. Nil for parameter nodes.
	Children []*TreeNode
	// Meta holds parameter metadata. Non-nil only for parameter nodes.
	Meta *ParameterMeta
	// Expanded tracks whether a folder is expanded in the TUI tree view.
	Expanded bool
}

// NewFolder creates a folder node with the given name and path.
func NewFolder(name, path string) *TreeNode {
	return &TreeNode{
		Name:     name,
		Path:     path,
		Type:     NodeFolder,
		Children: make([]*TreeNode, 0),
	}
}

// NewParameter creates a parameter leaf node with the given name, path, and metadata.
func NewParameter(name, path string, meta *ParameterMeta) *TreeNode {
	return &TreeNode{
		Name: name,
		Path: path,
		Type: NodeParameter,
		Meta: meta,
	}
}

// IsFolder returns true if the node is a folder.
func (n *TreeNode) IsFolder() bool {
	return n.Type == NodeFolder
}

// IsParameter returns true if the node is a parameter leaf.
func (n *TreeNode) IsParameter() bool {
	return n.Type == NodeParameter
}

// IsSecureString returns true if the node is a SecureString parameter.
// Always returns false for folder nodes.
func (n *TreeNode) IsSecureString() bool {
	return n.Meta != nil && n.Meta.Type == "SecureString"
}

// AddChild appends a child node and returns the parent for chaining.
func (n *TreeNode) AddChild(child *TreeNode) *TreeNode {
	n.Children = append(n.Children, child)
	return n
}

// FindChild returns the immediate child with the given name, or nil.
func (n *TreeNode) FindChild(name string) *TreeNode {
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// SortChildren recursively sorts children alphabetically with folders first.
func (n *TreeNode) SortChildren() {
	sort.Slice(n.Children, func(i, j int) bool {
		// Folders before parameters.
		if n.Children[i].Type != n.Children[j].Type {
			return n.Children[i].Type == NodeFolder
		}
		return n.Children[i].Name < n.Children[j].Name
	})
	for _, c := range n.Children {
		if c.IsFolder() {
			c.SortChildren()
		}
	}
}

// ChildCount returns the number of direct children.
func (n *TreeNode) ChildCount() int {
	return len(n.Children)
}

// ParameterCount returns the total number of parameter nodes in the subtree.
func (n *TreeNode) ParameterCount() int {
	if n.IsParameter() {
		return 1
	}
	count := 0
	for _, c := range n.Children {
		count += c.ParameterCount()
	}
	return count
}

// BuildTree constructs a tree from a flat list of parameter paths.
// Each path is expected to start with "/" (e.g. "/app/config/db_host").
// The root node represents "/" and all parameters are placed in the
// appropriate folder hierarchy.
func BuildTree(params []FlatParam) *TreeNode {
	root := NewFolder("/", "/")

	for _, p := range params {
		segments := splitPath(p.Path)
		current := root

		for i, seg := range segments {
			isLast := i == len(segments)-1

			if isLast {
				// Leaf parameter node.
				node := NewParameter(seg, p.Path, p.Meta)
				current.AddChild(node)
			} else {
				// Intermediate folder — find or create.
				child := current.FindChild(seg)
				if child == nil {
					folderPath := "/" + strings.Join(segments[:i+1], "/")
					child = NewFolder(seg, folderPath)
					current.AddChild(child)
				}
				current = child
			}
		}
	}

	root.SortChildren()
	return root
}

// FlatParam is a minimal representation of a parameter used as input to BuildTree.
type FlatParam struct {
	// Path is the full SSM parameter path (e.g. "/app/config/db_host").
	Path string
	// Meta holds optional parameter metadata.
	Meta *ParameterMeta
}

// splitPath splits an SSM path like "/app/config/db_host" into ["app", "config", "db_host"].
// Leading and trailing slashes are ignored; empty segments are skipped.
func splitPath(path string) []string {
	parts := strings.Split(path, "/")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

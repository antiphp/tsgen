package tsgen

import "strings"

// Package represents a Go package.
type Package struct {
	Name  string
	Nodes []Node
}

// Node represents a Go AST node.
type Node interface {
	GetName() string
}

// Supported AST nodes.
var (
	_ Node = (*NodeInterface)(nil)
	_ Node = (*NodeStruct)(nil)
	_ Node = (*NodeType)(nil)
)

// NodeInterface represents a Go interface.
type NodeInterface struct {
	Name string
	Doc  string

	// TODO: Add methods/functions.
}

func (n *NodeInterface) GetName() string {
	return n.Name
}

// NodeStruct represents a Go struct.
type NodeStruct struct {
	Name   string
	Doc    string
	Fields []*Field
}

func (n *NodeStruct) GetName() string {
	return n.Name
}

// NodeType represents a Go type alias (primitive or complex).
type NodeType struct {
	Name string
	Doc  string

	Type Type
}

func (n *NodeType) GetName() string {
	return n.Name
}

// Field represents a struct field.
type Field struct {
	Name string
	Doc  string
	Tags Tags

	Type Type
}

// Tags represents struct field tags.
type Tags map[string]string

// JSON checks whether the `json` tag contains the expected value.
func (t Tags) JSON(expect string) bool {
	raw, ok := t["json"]
	if !ok {
		return false
	}
	for _, elem := range strings.Split(raw, ",") {
		if strings.TrimSpace(elem) == expect {
			return true
		}
	}
	return false
}

// Type represents a Go type.
type Type interface{}

// Supported types.
var (
	_ Type = (*TypePointer)(nil)
	_ Type = (*TypePrimitive)(nil)
	_ Type = (*TypeMap)(nil)
	_ Type = (*TypeArray)(nil)
	_ Type = (*TypeReference)(nil)
	_ Type = (*TypeUnimplemented)(nil)
)

// TypePointer represents a pointer type.
type TypePointer struct {
	Type Type
}

// TypePrimitive represents a primitive type (string, int, bool, etc.).
type TypePrimitive struct {
	Name string
}

// TypeMap represents a map type.
type TypeMap struct {
	Key   Type
	Value Type
}

// TypeArray represents a slice/array type.
type TypeArray struct {
	Type Type
}

// TypeReference represents a reference to another package.
type TypeReference struct {
	PkgPath string
	Name    string
}

// TypeUnimplemented represents a type that is not yet implemented (e.g. func, chan).
type TypeUnimplemented struct {
	Name string
}

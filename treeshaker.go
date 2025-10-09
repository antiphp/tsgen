package tsgen

import (
	"slices"
)

// TreeShaker removes unused nodes from the AST based on entry packages.
type TreeShaker struct {
	EntryPkgs []string

	idx index
}

// NewTreeShaker creates a new TreeShaker with the given entry packages.
func NewTreeShaker(entryPkgs ...string) *TreeShaker {
	return &TreeShaker{
		EntryPkgs: entryPkgs,
	}
}

// Shake removes unused nodes from the given packages and returns the number of removed nodes and the new packages.
func (t *TreeShaker) Shake(pkgs []*Package) (int, []*Package) {
	t.index(pkgs)
	t.markUsed()

	unused, outPkgs := t.removeUnused(pkgs)
	return unused, outPkgs
}

// index maps nodes to their references and what references them.
type index map[indexKey]indexValue

// indexKey is a key for the index map.
type indexKey struct {
	Pkg  string
	Name string
}

// indexValue is a value for the index map.
type indexValue struct {
	Node         Node
	ReferencedBy []indexKey
}

func (t *TreeShaker) index(pkgs []*Package) {
	t.idx = make(index)

	for _, pkg := range pkgs {
		for _, node := range pkg.Nodes {
			key := indexKey{
				Pkg:  pkg.Name,
				Name: node.GetName(),
			}

			t.idx[key] = indexValue{
				Node: node,
			}
		}
	}
}

func (t *TreeShaker) markUsed() {
	for key := range t.idx {
		if !slices.Contains(t.EntryPkgs, key.Pkg) {
			continue
		}

		t.mark(key, indexKey{})
	}
}

func (t *TreeShaker) mark(key, referencedBy indexKey) {
	val := t.idx[key]
	val.ReferencedBy = append(val.ReferencedBy, referencedBy)

	t.idx[key] = val

	for _, ref := range val.Node.GetRefs() {
		t.mark(indexKey{ref.PkgPath, ref.Name}, key)
	}
}

func (t *TreeShaker) removeUnused(pkgs []*Package) (int, []*Package) {
	outPkgs := make([]*Package, 0, len(pkgs))
	var removed int

	for _, pkg := range pkgs {
		outPkg := &Package{
			Name: pkg.Name,
			Doc:  pkg.Doc,
		}

		nodes := make([]Node, 0, len(pkg.Nodes))
		for _, node := range pkg.Nodes {
			key := indexKey{pkg.Name, node.GetName()}
			val := t.idx[key]

			if len(val.ReferencedBy) == 0 {
				removed++
				continue
			}

			nodes = append(nodes, node)
		}

		outPkg.Nodes = nodes
		outPkgs = append(outPkgs, outPkg)
	}

	return removed, outPkgs
}

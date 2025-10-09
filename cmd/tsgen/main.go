package main

import (
	"fmt"
	"log"

	"github.com/antiphp/tsgen"
	_ "k8s.io/api/core/v1"
)

func main() {
	pkgNames := []string{
		"k8s.io/api/core/v1",
	}

	// Parse.

	parser := tsgen.NewParser(tsgen.Config{})
	pkgs, err := parser.Parse(pkgNames...)
	if err != nil {
		log.Fatal("parsing packages:", err)
	}

	// Shake off unused.

	treeShaker := tsgen.NewTreeShaker(pkgNames...)

	var gone int
	gone, pkgs = treeShaker.Shake(pkgs)

	fmt.Printf("Removed %d unused nodes\n", gone)

	// Resolve.
	// TODO.

	// Generate.
	// TODO.

	for _, pkg := range pkgs {
		fmt.Printf("Package: %s\n", pkg.Name)
		for _, node := range pkg.Nodes {
			fmt.Printf("  Node: %s (%T)\n", node.GetName(), node)
		}
	}
}

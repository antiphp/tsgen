package main

import (
	"fmt"
	"log"

	"github.com/antiphp/tsgen"
	_ "k8s.io/api/core/v1"
)

func main() {
	parser := tsgen.NewParser(tsgen.Config{})
	pkgs, err := parser.Parse("k8s.io/api/core/v1")
	if err != nil {
		log.Fatal("parsing packages:", err)
	}

	// Display.

	for _, pkg := range pkgs {
		fmt.Printf("Package: %s\n", pkg.Name)
		for _, node := range pkg.Nodes {
			fmt.Printf("  Node: %s (%T)\n", node.GetName(), node)
		}
	}
}

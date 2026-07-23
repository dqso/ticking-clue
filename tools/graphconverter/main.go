// Command graphconverter converts neo4j apoc JSON exports
// from the input directory into binary protobuf graphs.
package main

import (
	"flag"
	"log"
)

func main() {
	inputDir := flag.String("input", "tools/graphconverter/input", "directory with .json exports from neo4j")
	outputDir := flag.String("output", "tools/graphconverter/output", "directory for binary .pb graphs")
	flag.Parse()

	if err := run(*inputDir, *outputDir); err != nil {
		log.Fatal(err)
	}
}

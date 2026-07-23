package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/proto"
)

// run converts every .json export in inputDir into a .pb file in outputDir.
func run(inputDir, outputDir string) error {
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("read input dir: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		dst := strings.TrimSuffix(name, ".json") + ".pb"
		if err := convertFile(filepath.Join(inputDir, name), filepath.Join(outputDir, dst)); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

// convertFile parses one export and writes it as binary protobuf.
func convertFile(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	graph, st, err := readNeo4jGraph(f)
	if err != nil {
		return err
	}
	st.print(filepath.Base(src))

	data, err := proto.Marshal(graph)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(dst, data, 0o644)
}

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var (
		evalRecordsDir = flag.String("dir", "eval_records", "eval records root directory")
		keepReadme     = flag.Bool("keep-readme", true, "keep eval_records/README.md")
	)
	flag.Parse()

	entries, err := os.ReadDir(*evalRecordsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read eval_records failed: %v\n", err)
		os.Exit(1)
	}

	removed := 0
	for _, entry := range entries {
		name := entry.Name()
		if *keepReadme && name == "README.md" {
			continue
		}
		target := filepath.Join(*evalRecordsDir, name)
		if err := os.RemoveAll(target); err != nil {
			fmt.Fprintf(os.Stderr, "remove failed: %s: %v\n", target, err)
			os.Exit(1)
		}
		removed++
	}

	fmt.Println("ai_eval_clear success")
	fmt.Printf("removed entries: %d\n", removed)
	fmt.Printf("target dir: %s\n", *evalRecordsDir)
}

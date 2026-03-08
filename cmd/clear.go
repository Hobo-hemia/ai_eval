package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func runClear(args []string) int {
	fs := flag.NewFlagSet("clear", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		evalRecordsDir = fs.String("dir", "eval_records", "eval records root directory")
		keepReadme     = fs.Bool("keep-readme", true, "keep eval_records/README.md")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	entries, err := os.ReadDir(*evalRecordsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read eval_records failed: %v\n", err)
		return 1
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
			return 1
		}
		removed++
	}

	fmt.Println("ai_eval clear success")
	fmt.Printf("removed entries: %d\n", removed)
	fmt.Printf("target dir: %s\n", *evalRecordsDir)
	return 0
}

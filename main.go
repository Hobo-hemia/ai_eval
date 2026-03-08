package main

import (
	"os"

	"ai_eval/cmd"
)

func main() {
	os.Exit(cmd.Execute(os.Args[1:]))
}

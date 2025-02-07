package main

import (
	"fmt"
	"os"

	"what/internal/rules"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: gen_docs <filename>")
		os.Exit(1)
	}
	f, err := os.Create(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := rules.GenerateDocs(f); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "Documentation generated and saved to:", os.Args[1])
}

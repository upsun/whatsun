package main

import (
	"fmt"
	"os"
	"what/internal/rules"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: warm_cache <filename>")
		os.Exit(1)
	}
	if err := rules.WarmCache(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "Cache saved to:", os.Args[1])
}

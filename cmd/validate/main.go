package main

import (
	"fmt"
	"os"

	"github.com/upsun/whatsun/pkg/rules"
)

func main() {
	configDir := "config"
	if len(os.Args) > 1 {
		configDir = os.Args[1]
	}

	fmt.Fprintf(os.Stderr, "Validating rules configuration in %s directory...\n", configDir)

	_, err := rules.LoadFromYAMLDir(os.DirFS("."), configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "The rules configuration is valid.")
}

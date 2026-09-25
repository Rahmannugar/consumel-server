package main

import (
	"fmt"
	"os"

	"github.com/Rahmannugar/consumel-server/internal/openapi"
)

func main() {
	document, err := openapi.Document()
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile("internal/openapi/openapi.json", document, 0o644); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

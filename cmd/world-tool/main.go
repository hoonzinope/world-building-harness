package main

import (
	"os"

	"github.com/hoonzi/world-harness/internal/harness"
)

func main() {
	os.Exit(harness.Run(os.Args[1:]))
}

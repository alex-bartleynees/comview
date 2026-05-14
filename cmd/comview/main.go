package main

import (
	"fmt"
	"os"

	"github.com/rockorager/comview/tui"
)

func main() {
	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "comview: %v\n", err)
		os.Exit(1)
	}
}

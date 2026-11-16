package main

import (
	"fmt"
	"os"

	"github.com/nxkh4ng/gatekeep/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "gatekeep:", err)
		os.Exit(1)
	}
}

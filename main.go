package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		Usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "clone":
		err = Clone(args)
	case "pull":
		err = Pull(args)
	case "help", "--help", "-h":
		Usage()
	default:
		fmt.Fprintf(os.Stderr, "git-slot: unknown command %q\n", cmd)
		Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "git-slot: %v\n", err)
		os.Exit(1)
	}
}

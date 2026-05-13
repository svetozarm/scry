package main

import (
	"errors"
	"os"

	"github.com/svetozarm/scry/cmd"
)

var version string

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		var se interface{ ExitCode() int }
		if errors.As(err, &se) {
			os.Exit(se.ExitCode())
		}
		os.Exit(1)
	}
}

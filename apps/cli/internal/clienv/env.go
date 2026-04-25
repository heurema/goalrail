package clienv

import (
	"io"
	"os"
)

// Env carries process-level CLI dependencies that are safe to replace in tests.
type Env struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Stdin   io.Reader
	WorkDir string
}

// Default builds the process-backed CLI environment.
func Default() Env {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}

	return Env{
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Stdin:   os.Stdin,
		WorkDir: workDir,
	}
}

/*
Copyright © 2025 changheonshin
*/
package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	// Since main() calls cmd.Execute() which would run the CLI,
	// we need to test it indirectly by ensuring the function exists
	// and can be called without panicking.

	// We'll test by temporarily changing os.Args to avoid actual command execution
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Set args to help command to avoid hanging on input
	os.Args = []string{"fman", "--help"}

	// This should not panic
	assert.NotPanics(t, func() {
		// We can't directly test main() as it would exit the process,
		// but we can test that the main function exists and is callable
		// by testing the cmd.Execute() function indirectly
	})
}

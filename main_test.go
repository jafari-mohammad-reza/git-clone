package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func fetchStdOut(t *testing.T) string {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to get reader and writer: %s", err.Error())
	}

	os.Stdout = w
	main()
	w.Close()
	os.Stdout = oldStdout
	buf := new(strings.Builder)
	_, err = io.Copy(buf, r)
	if err != nil {
		t.Fatalf("failed to copy stdout to buffer")
	}
	output := buf.String()

	return output
}
func fetchStdErr(t *testing.T) string {
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to get reader and writer: %s", err.Error())
	}

	os.Stderr = w
	main()
	w.Close()
	os.Stderr = oldStderr
	buf := new(strings.Builder)
	_, err = io.Copy(buf, r)
	if err != nil {
		t.Fatalf("failed to copy stdErr to buffer")
	}
	output := buf.String()

	return output
}
func TestCommands(t *testing.T) {
	t.Run("should print help command", func(t *testing.T) {
		os.Args = []string{
			"", // for first input
			"help",
		}
		output := fetchStdOut(t)
		if !strings.ContainsAny(output, "list of possible commands") {
			t.Fatalf("help command was not printed")
		}
	})
	t.Run("should print help command for read-tree", func(t *testing.T) {
		os.Args = []string{
			"", // for first input
			"help",
			"read-tree",
		}
		output := fetchStdOut(t)

		if !strings.ContainsAny(output, "read-tree <hash>: reads the changes of a hash and prints the changes content") {
			t.Fatalf("read tree help content is not printed")
		}
	})
	t.Run("should print error for invalid subcommand", func(t *testing.T) {
		os.Args = []string{
			"", // for first input
			"help",
			"invalid-sub",
		}
		output := fetchStdErr(t)

		if !strings.ContainsAny(output, "invalid sub command 'invalid-sub' use 'help' for list of possible commands") {
			t.Fatalf("invalid sub command message was not printed")
		}
	})
}

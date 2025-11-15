package main

import (
	"fmt"
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
	t.Run("should print help command for cat-file", func(t *testing.T) {
		os.Args = []string{
			"", // for first input
			"help",
			"cat-file",
		}
		output := fetchStdOut(t)

		if !strings.ContainsAny(output, "cat-file <hash>: reads the changes of a hash and prints the changes content") {
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
func TestInitCommand(t *testing.T) {
	t.Run("should initialize in tmp dir", func(t *testing.T) {
		t.Setenv("run_env", "test")
		initialize()
		_, err := os.Stat("tmp/")
		if err != nil {
			t.Fatalf("tmp dir does not exist: %s", err.Error())
		}
		_, err = os.Stat("tmp/.git")
		if err != nil {
			t.Fatalf(".git dir does not exist: %s", err.Error())
		}

		_, err = os.Stat("tmp/.git/objects")
		if err != nil {
			t.Fatalf(".git/objects dir does not exist: %s", err.Error())
		}

		_, err = os.Stat("tmp/.git/refs")
		if err != nil {
			t.Fatalf(".git/refs dir does not exist: %s", err.Error())
		}
		head, err := os.ReadFile("tmp/.git/HEAD")
		if err != nil {
			t.Fatalf("HEAD file does not exist")
		}

		if !strings.ContainsAny(string(head), "ref: refs/heads/main") {
			t.Fatal("the ref content was wrong")
		}

		t.Cleanup(func() {
			if err := os.RemoveAll("tmp/"); err != nil {
				t.Fatalf("failed to remove tmp dir at end: %s", err.Error())
			}
		})
	})
	t.Run("should return already exist err", func(t *testing.T) {
		initialize()
		err := fetchStdErr(t)
		if !strings.ContainsAny(err, "the .git dir already exists") {
			t.Fatal("the .git dir already exists message was not printed")
		}
	})
}

func TestCatFile(t *testing.T) {
	t.Run("should find the file with the prfix", func(t *testing.T) {
		t.Setenv("run_env", "test")
		initialize()
		keyname := "abcdefg"
		if err := os.MkdirAll(fmt.Sprintf("tmp/.git/objects/%s", keyname[:2]), 0755); err != nil {
			t.Fatalf("failed to create the dir with given prefix: %s", err.Error())
		}
		if err := os.WriteFile(fmt.Sprintf("tmp/.git/objects/%s/%s", keyname[:2], keyname[2:]), []byte("test"), 0755); err != nil {
			t.Fatalf("failed to write file with keyname after second char: %s", err.Error())
		}
		catFile([]string{
			"",
			"",
			keyname,
		})
		err := fetchStdErr(t)

		if strings.HasPrefix(err, "cat-file err:") {
			t.Fatal("stderr is not empty")
		}
		// TODO: validate the blob written to the object file
		t.Cleanup(func() {
			if err := os.RemoveAll("tmp/"); err != nil {
				t.Fatalf("failed to cleanup tmp dir: %s", err.Error())
			}
		})
	})
	t.Run("should throw error that inpt is not given", func(t *testing.T) {
		catFile([]string{
			"",
			"",
		})
		err := fetchStdErr(t)
		if !strings.ContainsAny(err, "cat-file err: specify which file to read use 'help cat-file' for more info") {
			t.Fatalf("expected to get 'cat-file err: specify which file to read use 'help cat-file' for more info' but got %s", err)
		}

	})
	t.Run("should throw error that objects dir is not found", func(t *testing.T) {
		t.Setenv("run_env", "test")
		initialize()
		keyname := "abcdefg"

		catFile([]string{
			"",
			"",
			keyname,
		})
		err := fetchStdErr(t)
		if !strings.ContainsAny(err, "cat-file err: the abcdefg reference dir does not exits in tmp/.git/objects at tmp/.git/objects/ab") {
			t.Fatal("wrong error message")
		}
		t.Cleanup(func() {
			if err := os.RemoveAll("tmp/"); err != nil {
				t.Fatalf("failed to cleanup tmp dir: %s", err.Error())
			}
		})
	})

	t.Run("should throw error that the given file is not found from second char to last", func(t *testing.T) {
		t.Setenv("run_env", "test")
		initialize()
		keyname := "abcdefg"
		if err := os.MkdirAll(fmt.Sprintf("tmp/.git/objects/%s", keyname[:2]), 0755); err != nil {
			t.Fatalf("failed to create the dir with given prefix: %s", err.Error())
		}

		catFile([]string{
			"",
			"",
			keyname,
		})
		err := fetchStdErr(t)

		if !strings.ContainsAny(err, "cat-file err: failed to find any reference with prefix of : cdefg") {
			t.Fatal("stderr is not empty")
		}
		t.Cleanup(func() {
			if err := os.RemoveAll("tmp/"); err != nil {
				t.Fatalf("failed to cleanup tmp dir: %s", err.Error())
			}
		})
	})
}

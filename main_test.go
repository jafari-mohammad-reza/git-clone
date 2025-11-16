package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestInitCommand(t *testing.T) {
	t.Run("should initialize in tmp dir", func(t *testing.T) {
		t.Setenv("run_env", "test")
		initialize("test")
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
		if err := initialize("test"); err != nil {
			t.Fatalf("initialize got error: %s", err.Error())
		}
	})
}

func TestCatFile(t *testing.T) {
	t.Run("should find the file with the prfix", func(t *testing.T) {
		t.Setenv("run_env", "test")
		initialize("test")
		keyname := "abcdefg"
		if err := os.MkdirAll(fmt.Sprintf("tmp/.git/objects/%s", keyname[:2]), 0755); err != nil {
			t.Fatalf("failed to create the dir with given prefix: %s", err.Error())
		}
		if err := os.WriteFile(fmt.Sprintf("tmp/.git/objects/%s/%s", keyname[:2], keyname[2:]), []byte("test"), 0755); err != nil {
			t.Fatalf("failed to write file with keyname after second char: %s", err.Error())
		}
		hash, err := hashObject("main.go")
		if err != nil {
			t.Fatalf("hash object error: %s", err.Error())
		}

		file, err := catFile(hash)
		fmt.Printf("file: %v\n", file)

		if err != nil {
			t.Fatalf("cat file error: %s", err.Error())
		}
		t.Cleanup(func() {
			if err := os.RemoveAll("tmp/"); err != nil {
				t.Fatalf("failed to cleanup tmp dir: %s", err.Error())
			}
		})
	})

	t.Run("should throw error that objects dir is not found", func(t *testing.T) {
		t.Setenv("run_env", "test")
		initialize("test")
		keyname := "abcdefg"

		_, err := catFile(keyname)
		if !strings.ContainsAny(err.Error(), "cat-file err: the abcdefg reference dir does not exits in tmp/.git/objects at tmp/.git/objects/ab") {
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
		initialize("test")
		keyname := "abcdefg"
		if err := os.MkdirAll(fmt.Sprintf("tmp/.git/objects/%s", keyname[:2]), 0755); err != nil {
			t.Fatalf("failed to create the dir with given prefix: %s", err.Error())
		}

		_, err := catFile(keyname)

		if !strings.ContainsAny(err.Error(), "cat-file err: failed to find any reference with prefix of : cdefg") {
			t.Fatal("stderr is not empty")
		}
		t.Cleanup(func() {
			if err := os.RemoveAll("tmp/"); err != nil {
				t.Fatalf("failed to cleanup tmp dir: %s", err.Error())
			}
		})
	})
}

func TestLogCommand(t *testing.T) {
	// TODO‌: need implementing write hash for testing manually
	t.Run("should return list of commits/tree", func(t *testing.T) {})
	t.Run("should fail to read commit file", func(t *testing.T) {})
	t.Run("should ignore blob file", func(t *testing.T) {})

}

func TestHashObject(t *testing.T) {
	t.Run("should hash file successfully adn read it back to normal", func(t *testing.T) {
		t.Setenv("run_env", "test")
		initialize("test")

		hash, err := hashObject("main.go")
		if err != nil {
			t.Fatalf("hash object throw error: %s", err.Error())
		}
		if hash == "" {
			t.Fatal("unexpected hash: empty hash string")
		}

		hashPath := fmt.Sprintf("tmp/.git/objects/%s/%s", hash[:2], hash[2:])
		_, err = os.ReadFile(hashPath)
		if err != nil {
			t.Fatalf("failed to read %s: %s", hashPath, err.Error())
		}

		catFile(hash)

		t.Cleanup(func() {
			if err := os.RemoveAll("tmp/"); err != nil {
				t.Fatalf("failed to remove tmp for cleanup: %s", err.Error())
			}
		})
	})
}

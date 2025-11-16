package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
func TestListObjects(t *testing.T) {
	os.RemoveAll("tmp")
	defer os.RemoveAll("tmp")

	objDir := "tmp/.git/objects/ab"
	if err := os.MkdirAll(objDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	content := []byte("hello world\n")
	header := fmt.Sprintf("blob %d\x00", len(content))
	gitObj := append([]byte(header), content...)

	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(gitObj); err != nil {
		t.Fatalf("failed to write zlib: %v", err)
	}
	zw.Close()

	err := os.WriteFile("tmp/.git/objects/ab/cdef01", buf.Bytes(), 0644)
	if err != nil {
		t.Fatalf("failed to write git object: %v", err)
	}

	out, err := listObjects("test")
	if err != nil {
		t.Fatalf("listObjects err: %v", err)
	}

	if !strings.Contains(out, "blob - abcdef01") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestLsTree(t *testing.T) {
	os.RemoveAll("tmp")
	defer os.RemoveAll("tmp")

	dir := "tmp/.git/objects/ab"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	mode1 := "40000"
	name1 := "src"
	sha1_1 := bytes.Repeat([]byte{0x11}, 20)

	mode2 := "100644"
	name2 := "main.go"
	sha1_2 := bytes.Repeat([]byte{0x22}, 20)

	payload := []byte{}
	payload = append(payload, []byte(fmt.Sprintf("%s %s", mode1, name1))...)
	payload = append(payload, 0x00)
	payload = append(payload, sha1_1...)

	payload = append(payload, []byte(fmt.Sprintf("%s %s", mode2, name2))...)
	payload = append(payload, 0x00)
	payload = append(payload, sha1_2...)

	header := []byte(fmt.Sprintf("tree %d\x00", len(payload)))

	final := append(header, payload...)

	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(final); err != nil {
		t.Fatalf("zlib write failed: %v", err)
	}
	zw.Close()

	gitObjectHash := "ab" + "cdef01"
	filePath := "tmp/.git/objects/ab/cdef01"

	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	out, err := lsTree(gitObjectHash, "test")
	if err != nil {
		t.Fatalf("lsTree failed: %v", err)
	}

	expected := "src/\nmain.go\n"

	if out != expected {
		t.Fatalf("unexpected output:\n%s\nexpected:\n%s", out, expected)
	}
}
func TestWriteTree(t *testing.T) {
	tmp, err := os.MkdirTemp("", "git-tree-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(".git/objects", 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("a.txt", []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir("dir", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("dir/b.txt", []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, err := writeTree(".")
	if err != nil {
		t.Fatalf("writeTree failed: %v", err)
	}

	if !regexp.MustCompile("^[0-9a-f]{40}$").MatchString(hash) {
		t.Fatalf("invalid hash returned: %s", hash)
	}

	objPath := filepath.Join(".git", "objects", hash[:2], hash[2:])
	if _, err := os.Stat(objPath); err != nil {
		t.Fatalf("tree object not created at %s", objPath)
	}

	blobHash := func(data []byte) string {
		hdr := []byte("blob " + fmt.Sprint(len(data)) + "\x00")
		raw := append(hdr, data...)
		sum := sha1.Sum(raw)
		return hex.EncodeToString(sum[:])
	}

	aBlob := blobHash([]byte("hello"))
	bBlob := blobHash([]byte("world"))

	aBlobPath := filepath.Join(".git", "objects", aBlob[:2], aBlob[2:])
	bBlobPath := filepath.Join(".git", "objects", bBlob[:2], bBlob[2:])

	if _, err := os.Stat(aBlobPath); err != nil {
		t.Fatalf("blob for a.txt not created: %s", aBlobPath)
	}
	if _, err := os.Stat(bBlobPath); err != nil {
		t.Fatalf("blob for dir/b.txt not created: %s", bBlobPath)
	}

	data, err := os.ReadFile(objPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Fatalf("tree object empty or corrupted")
	}
}

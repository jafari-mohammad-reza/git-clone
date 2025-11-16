package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	args := os.Args
	if len(args) < 1 {
		fmt.Println("please specify you command.use help for list of commands")
		return
	}
	runEnv := os.Getenv("run_env")
	command := args[1]
	switch strings.ToLower(command) {
	case "help":
		subCmd := ""
		if len(args) >= 3 {
			subCmd = args[2]
		}
		resp, err := help(subCmd)
		if err != nil {
			println(err.Error())
			return
		}
		println(resp)
	case "init":
		if err := initialize(runEnv); err != nil {
			println(err.Error())
			return
		}
	case "cat-file":
		if len(args) < 3 {
			fmt.Println("give the hash you want to read")
			return
		}
		hash := args[2]
		resp, err := catFile(hash)
		if err != nil {
			println(err.Error())
			return
		}
		println(resp)
	case "hash-object":
		if len(args) < 3 {
			fmt.Println("give the file you want to hash")
			return
		}
		file := args[2]
		hash, err := hashObject(file)
		if err != nil {
			println(err.Error())
			return
		}
		println(hash)
	case "log":
		logs, err := log()
		if err != nil {
			println(err.Error())
			return
		}
		for _, log := range logs {
			println(log)
		}
	case "ls-objects":
		// this is not an actual git command just for practice
		// iot will read all hash directories in .git/objects and print each one with it object type
		resp, err := listObjects(runEnv)
		if err != nil {
			println(err.Error())
			return
		}
		println(resp)
	case "ls-tree":
		if len(args) < 3 {
			fmt.Println("give the hash you want to read")
			return
		}
		hash := args[2]
		resp, err := lsTree(hash, runEnv)
		if err != nil {
			println(err.Error())
			return
		}
		println(resp)
	case "write-tree":

		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		resp, err := writeTree(pwd)
		if err != nil {
			println(err.Error())
			return
		}
		println(resp)
	case "commit-tree":
		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		hash, err := writeTree(pwd)
		if err != nil {
			println(err.Error())
			return
		}

		if len(args) < 3 {
			fmt.Println("give the file you want to hash")
			return
		}
		msg := args[2]
		commitHash, err := commitTree(hash, msg)
		if err != nil {
			println(err.Error())
			return
		}
		println(commitHash)
	default:
		fmt.Printf("invalid command '%s' use help for list of commands\n", command)
	}

}
func commitTree(treeHash, msg string) (string, error) {
	raw := fmt.Sprintf("tree %s\n"+
		"parent %s\n"+
		"author teyyubismayil <tismayilov@pro.simbrella.com> 1661410769 +0400\n"+
		"teyyubismayil <tismayilov@pro.simbrella.com> 1661410769 +0400\n\n"+
		"%s\n", treeHash, "master", msg) // master for now

	header := fmt.Sprintf("blob %d\x00", len(raw))
	full := append([]byte(header), raw...)
	hasher := sha1.New()
	if _, err := hasher.Write(full); err != nil {
		return "", fmt.Errorf("failed to hash header: %s",  err.Error())

	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	targetDir := fmt.Sprintf(".git/objects/%s", hash[:2])
	run_env := os.Getenv("run_env")
	if run_env == "test" {
		targetDir = fmt.Sprintf("tmp/.git/objects/%s", hash[:2])
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create %s dir: %s", targetDir, err.Error())

	}
	out, err := os.OpenFile(fmt.Sprintf("%s/%s", targetDir, hash[2:]), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create %s file: %s", fmt.Sprintf("%s/%s", targetDir, hash[2:]), err.Error())

	}
	defer out.Close()
	writer := zlib.NewWriter(out)
	if _, err := writer.Write(full); err != nil {
		return "", fmt.Errorf("failed to write compressed data: %s", err.Error())

	}
	defer writer.Close()
	return hash , nil
}
func writeTree(dirPath string) (string, error) {
	type entry struct {
		name string
		data []byte
	}

	hexToRaw := func(h string) ([]byte, error) {
		b, err := hex.DecodeString(h)
		if err != nil {
			return nil, err
		}
		if len(b) != 20 {
			return nil, fmt.Errorf("sha length != 20 for %s", h)
		}
		return b, nil
	}

	writeObject := func(kind string, data []byte) (string, error) {
		header := fmt.Sprintf("%s %d\x00", kind, len(data))
		store := append([]byte(header), data...)
		sha := sha1.Sum(store)
		shaHex := hex.EncodeToString(sha[:])

		objDir := filepath.Join(".git", "objects", shaHex[:2])
		objPath := filepath.Join(objDir, shaHex[2:])

		if _, err := os.Stat(objPath); err == nil {
			return shaHex, nil
		}

		if err := os.MkdirAll(objDir, 0o755); err != nil {
			return "", fmt.Errorf("mkdir: %w", err)
		}

		f, err := os.Create(objPath)
		if err != nil {
			return "", fmt.Errorf("create: %w", err)
		}
		defer f.Close()

		zw := zlib.NewWriter(f)
		if _, err := zw.Write(store); err != nil {
			return "", fmt.Errorf("zlib write: %w", err)
		}
		if err := zw.Close(); err != nil {
			return "", fmt.Errorf("zlib close: %w", err)
		}

		return shaHex, nil
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", fmt.Errorf("read dir %s: %w", dirPath, err)
	}

	ignorePath := filepath.Join(dirPath, ".gitignore")
	ignores := map[string]bool{}
	if data, err := os.ReadFile(ignorePath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				ignores[line] = true
			}
		}
	}

	var treeEntries []entry

	for _, e := range entries {
		name := e.Name()
		if name == ".git" {
			continue
		}
		if ignores[name] {
			continue
		}

		full := filepath.Join(dirPath, name)

		if e.IsDir() {
			childHash, err := writeTree(full)
			if err != nil {
				return "", fmt.Errorf("subtree %s: %w", full, err)
			}
			raw, err := hexToRaw(childHash)
			if err != nil {
				return "", err
			}
			prefix := []byte(fmt.Sprintf("40000 %s\x00", name))
			treeEntries = append(treeEntries, entry{name, append(prefix, raw...)})
			continue
		}

		data, err := os.ReadFile(full)
		if err != nil {
			return "", fmt.Errorf("read file %s: %w", full, err)
		}

		blobHash, err := writeObject("blob", data)
		if err != nil {
			return "", fmt.Errorf("write blob %s: %w", full, err)
		}

		raw, err := hexToRaw(blobHash)
		if err != nil {
			return "", err
		}

		prefix := []byte(fmt.Sprintf("100644 %s\x00", name))
		treeEntries = append(treeEntries, entry{name, append(prefix, raw...)})
	}

	sort.Slice(treeEntries, func(i, j int) bool {
		return treeEntries[i].name < treeEntries[j].name
	})

	var content []byte
	for _, te := range treeEntries {
		content = append(content, te.data...)
	}

	treeHash, err := writeObject("tree", content)
	if err != nil {
		return "", fmt.Errorf("write tree: %w", err)
	}

	return treeHash, nil
}

func lsTree(hash, runEnv string) (string, error) {
	objDir := ".git/objects"
	if runEnv == "test" {
		objDir = "tmp/.git/objects"
	}
	filePath := path.Join(objDir, hash[:2], hash[2:])
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to open %s: %s", filePath, err.Error())
	}
	defer file.Close()

	reader, err := zlib.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("failed to read refPath: %s", err.Error())
	}
	defer reader.Close()

	buf := bufio.NewReader(reader)

	header, err := buf.ReadBytes(0x00)
	if err != nil {
		return "", fmt.Errorf("failed to read null byte: %s", err.Error())
	}

	header = header[:len(header)-1]
	parts := bytes.SplitN(header, []byte(" "), 2)
	objSize, _ := strconv.Atoi(string(parts[1]))
	payload := make([]byte, objSize)
	_, err = io.ReadFull(buf, payload)
	if err != nil {
		return "", fmt.Errorf("failed to read buf into payload: %s", err.Error())
	}
	i := 0
	resp := ""
	for i < len(payload) {
		j := bytes.IndexByte(payload[i:], ' ')
		if j < 0 {
			return "", fmt.Errorf("invalid tree: missing space for mode")
		}
		mode := string(payload[i : i+j])
		i += j + 1

		k := bytes.IndexByte(payload[i:], 0x00)
		if k < 0 {
			return "", fmt.Errorf("invalid tree: missing null after filename")
		}
		filename := string(payload[i : i+k])
		i += k + 1

		if i+20 > len(payload) {
			return "", fmt.Errorf("invalid tree: truncated sha1")
		}
		i += 20

		if mode == "40000" {
			filename = filename + "/"
		}
		resp = resp + strings.Trim(filename, " ") + "\n"
	}

	return resp, nil
}

func listObjects(run_env string) (string, error) {
	targetDir := ".git/objects"
	if run_env == "test" {
		targetDir = "tmp/.git/objects"
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return "", fmt.Errorf("failed to read %s dir entries: %s", targetDir, err.Error())
	}

	notSearch := []string{
		"info",
		"pack",
	}
	resp := ""
	for _, entry := range entries {
		if slices.Contains(notSearch, entry.Name()) {
			continue
		}
		if entry.IsDir() {
			subEntries, err := os.ReadDir(path.Join(targetDir, entry.Name()))
			if err != nil {
				return "", fmt.Errorf("failed to read sub entry content for %s dir: %s", entry.Name(), err.Error())
			}
			for _, subEntry := range subEntries {

				refPath := fmt.Sprintf("%s/%s/%s", targetDir, entry.Name(), subEntry.Name())
				file, err := os.OpenFile(refPath, os.O_RDONLY, 0755)
				if err != nil {
					return "", fmt.Errorf("failed to open file: %s", err.Error())
				}
				defer file.Close()

				reader, err := zlib.NewReader(file)
				if err != nil {
					return "", fmt.Errorf("failed to read refPath: %s", err.Error())
				}
				defer reader.Close()

				buf := bufio.NewReader(reader)

				header, err := buf.ReadBytes(0x00)
				if err != nil {
					return "", fmt.Errorf("failed to read null byte: %s", err.Error())
				}

				header = header[:len(header)-1]

				parts := bytes.SplitN(header, []byte(" "), 2)
				objType := string(parts[0])
				resp = resp + "\n" + fmt.Sprintf("%s - %s", objType, entry.Name()+subEntry.Name())
			}
		}

	}
	return resp, nil
}

func hashObject(file string) (string, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %s", file, err.Error())

	}
	header := fmt.Sprintf("blob %d\x00", len(raw))
	full := append([]byte(header), raw...)
	hasher := sha1.New()
	if _, err = hasher.Write(full); err != nil {
		return "", fmt.Errorf("failed to hash header %s: %s", file, err.Error())

	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	targetDir := fmt.Sprintf(".git/objects/%s", hash[:2])
	run_env := os.Getenv("run_env")
	if run_env == "test" {
		targetDir = fmt.Sprintf("tmp/.git/objects/%s", hash[:2])
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create %s dir: %s", targetDir, err.Error())

	}
	out, err := os.OpenFile(fmt.Sprintf("%s/%s", targetDir, hash[2:]), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create %s file: %s", fmt.Sprintf("%s/%s", targetDir, hash[2:]), err.Error())

	}
	defer out.Close()
	writer := zlib.NewWriter(out)
	if _, err := writer.Write(full); err != nil {
		return "", fmt.Errorf("failed to write compressed data: %s", err.Error())

	}
	defer writer.Close()

	return hash, nil
}

// NOTE: only commit and tree messages are handled
func log() ([]string, error) {
	run_env := os.Getenv("run_env")

	searchDir := ".git/objects"
	if run_env == "test" {
		searchDir = "tmp/.git/objects"
	}

	notSearch := []string{
		"info",
		"pack",
	}

	stat, err := os.Stat(searchDir)
	if err != nil {
		return nil, fmt.Errorf("log err: the %s dir does not exist", searchDir)

	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("log err: the %s is not a dir", searchDir)

	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil, fmt.Errorf("log err: failed to read %s entries", searchDir)

	}

	refFiles := make([]string, 0)

	for _, entry := range entries {
		if slices.Contains(notSearch, entry.Name()) {
			continue
		}

		if !entry.IsDir() || len(entry.Name()) != 2 {
			continue
		}

		subDir := filepath.Join(searchDir, entry.Name())
		subEntries, err := os.ReadDir(subDir)
		if err != nil {
			return nil, fmt.Errorf("log err: failed to read %s entries: %s", entry.Name(), err.Error())

		}

		for _, subEntry := range subEntries {
			if subEntry.IsDir() {
				continue
			}

			sha := entry.Name() + subEntry.Name()
			if len(sha) != 40 {
				continue
			}

			refFiles = append(refFiles, sha)
		}
	}
	responses := make([]string, 0, len(refFiles))
	for _, ref := range refFiles {
		path := filepath.Join(searchDir, ref[:2], ref[2:])

		objType, authorName, authorEmail, timestamp, msg, err :=
			readCommitRef(path)

		if err != nil {
			continue
		}

		responses = append(responses, fmt.Sprintf("%s %s\nAuthor: %s %s\nDate: %s\n\n%s\n\n",
			objType, ref, authorName, authorEmail, timestamp, msg))
	}
	return responses, nil
}

func readCommitRef(refPath string) (string, string, string, string, string, error) {
	file, err := os.OpenFile(refPath, os.O_RDONLY, 0755)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("failed to open file: %s", err.Error())
	}
	defer file.Close()

	reader, err := zlib.NewReader(file)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("failed to read refPath: %s", err.Error())
	}
	defer reader.Close()

	buf := bufio.NewReader(reader)

	header, err := buf.ReadBytes(0x00)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("failed to read null byte: %s", err.Error())
	}

	header = header[:len(header)-1]

	parts := bytes.SplitN(header, []byte(" "), 2)
	objType := string(parts[0])
	allowedTypes := []string{
		"commit",
		"tree",
	}
	if !slices.Contains(allowedTypes, objType) {
		return "", "", "", "", "", errors.New("none commit ref")
	}

	objSize, _ := strconv.Atoi(string(parts[1]))

	payload := make([]byte, objSize)
	_, err = io.ReadFull(buf, payload)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("failed to read buf into payload: %s", err.Error())
	}

	payloadStr := string(payload)
	lines := strings.Split(payloadStr, "\n")

	var authorName, authorEmail, timestamp string
	var msg string

	for i := range lines {
		line := lines[i]

		if strings.HasPrefix(line, "author ") {
			fields := strings.Fields(line)
			authorName = fields[1]
			authorEmail = strings.Trim(fields[2], "<>")
			timestamp = fields[3]
			continue
		}

		if line == "" && i < len(lines)-1 {
			msg = strings.Join(lines[i+1:], "\n")
			break
		}
	}

	tsInt, err := strconv.Atoi(timestamp)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("failed to parse timestamp of %s: %s", timestamp, err.Error())
	}

	t := time.Unix(int64(tsInt), 0).Local().String()
	return objType, authorName, authorEmail, t, msg, nil
}

func catFile(inp string) (string, error) {
	run_env := os.Getenv("run_env")

	searchDir := ".git/objects"
	if run_env == "test" {
		searchDir = "tmp/.git/objects"
	}

	objDir := fmt.Sprintf("%s/%s", searchDir, inp[:2])

	stat, err := os.Stat(objDir)
	if err != nil {
		return "", fmt.Errorf("cat-file err: the %s reference dir does not exits in %s at %s", inp, searchDir, objDir)
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("cat-file err: the reference is not a dir")
	}
	entries, err := os.ReadDir(objDir)
	if err != nil {
		return "", fmt.Errorf("cat-file err: failed to read %s entries: %s", objDir, err.Error())
	}
	found := ""
	for _, entry := range entries {
		entry.Name()
		if strings.HasPrefix(entry.Name(), inp[2:]) {
			found = entry.Name()
			break
		}
	}
	if found == "" {
		return "", fmt.Errorf("cat-file err: failed to find any reference with prefix of : %s", inp[2:])

	}
	rp := fmt.Sprintf("%s/%s/%s", searchDir, inp[:2], found)

	_, err = os.Stat(rp)
	if err != nil {
		return "", fmt.Errorf("cat-file err: the %s reference does not exits in %s at %s", inp, searchDir, rp)

	}
	refFile, err := os.OpenFile(rp, os.O_RDONLY, 0755)
	if err != nil {
		return "", fmt.Errorf("cat-file err: failed to open %s file: %s", rp, err.Error())

	}
	reader, err := zlib.NewReader(refFile)
	if err != nil {
		return "", fmt.Errorf("cat-file err: failed to create reader for %s file in zlib: %s", rp, err.Error())

	}
	defer reader.Close()
	buf := bufio.NewReader(reader)
	header, err := buf.ReadBytes(0x00) // read until null byte
	if err != nil {
		panic(err)
	}
	header = header[:len(header)-1] // remove last part which is null byte
	parts := bytes.SplitN(header, []byte(" "), 2)
	objType := string(parts[0])
	objSize, _ := strconv.Atoi(string(parts[1]))
	payload := make([]byte, objSize)
	_, err = io.ReadFull(buf, payload)
	if err != nil {
		return "", fmt.Errorf("cat-file err: failed to read payload of %s object type: %s", err.Error(), objType)

	}
	return fmt.Sprintf("%s payload is:\n%s%s%s", rp, Blue, string(payload), Reset), nil // TODO: make the outcome look better

}

func initialize(runEnv string) error {
	initDir := ".git"
	if runEnv == "test" {
		initDir = "tmp/.git"
	}
	_, err := os.Stat(initDir)
	if err != nil {
		if err := os.MkdirAll(initDir, 0755); err != nil {
			return fmt.Errorf("failed to create .git dir: %s", err.Error())
		}
		if err := os.MkdirAll(fmt.Sprintf("%s/objects", initDir), 0755); err != nil {
			return fmt.Errorf("failed to create %s/objects dir: %s", initDir, err.Error())
		}
		if err := os.MkdirAll(fmt.Sprintf("%s/refs", initDir), 0755); err != nil {
			return fmt.Errorf("failed to create %s/refs dir: %s", initDir, err.Error())
		}
		if err := os.MkdirAll(fmt.Sprintf("%s/logs", initDir), 0755); err != nil {
			return fmt.Errorf("failed to create %s/logs dir: %s", initDir, err.Error())
		}
		if err := os.WriteFile(fmt.Sprintf("%s/HEAD", initDir), []byte("ref: refs/heads/main\n"), 0755); err != nil {
			return fmt.Errorf("failed to write %s/HEAD file: %s", initDir, err.Error())
		}
	} else {
		println("the .git dir already exists")
	}
	return nil
}

func help(subCmd string) (string, error) {
	if subCmd != "" {
		switch subCmd {
		case "init":
			return `
				init: initialized .git directory along:
					-  .git/objects => for storing hash files
					-  .git/refs => for storing refs(branches) (wont implement this one in clone)
					- .git/HEAD => the file for storing the head commit of current branch
			`, nil
		case "hash-object":
			return "hash-object <file>: creates a hash of file using its size and blob content then store that hash in .git/objects/{fist two char of hash}/{from second character to end of hash} and then write the compressed content by suing zlib to the hash file", nil
		case "cat-file":
			return "cat-file <hash>: reads the changes of a hash and prints the changes content", nil
		case "log":
			return "log: prints commits and tree hashes. and shows its author, date of commit and the commit message", nil
		case "ls-objects":
			return "ls-objects: use for list the objects stored ar .git/objects with their type", nil
		case "ls-tree":
			return "ls-tree: give a hash and see list of files in that tree", nil
		case "write-tree":
			return "write-tree => creates a tree object from the current state of the staging area", nil
		default:
			return "", fmt.Errorf("invalid sub command '%s' use 'help' for list of possible commands", subCmd)
		}
	} else {
		return `
			init => initialize required files and directories. 
			cat-file => read a ref compressed hash file into actual content.
			hash-object => read a ref compressed hash file into actual content.
			log => shows list commits.
			ls-objects => *NOT AN OFFICIAL COMMAND* use for list the objects stored ar .git/objects with their type
			ls-tree => give a hash and see list of files in that tree
			write-tree => creates a tree object from the current state of the staging area(current dir)
		`, nil
	}
}

const (
	White = "\033[37m"
	Blue  = "\033[34m"
	Red   = "\033[31m"
	Reset = "\033[0m"
)

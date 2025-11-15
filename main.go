package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

func main() {
	args := os.Args
	if len(args) < 1 {
		respErr("please specify you command.use help for list of commands")
		return
	}
	command := args[1]
	switch strings.ToLower(command) {
	case "help":
		help(args)
	case "init":
		initialize()
	case "cat-file":
		catFile(args)
	case "log":
		log()
	default:
		respErrF("invalid command '%s' use help for list of commands", command)
	}

}
func log() {
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
		respErrF("log err: the %s dir does not exist", searchDir)
		return
	}
	if !stat.IsDir() {
		respErrF("log err: the %s is not a dir", searchDir)
		return
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		respErrF("log err: failed to read %s entries", searchDir)
		return
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
			respErrF("log err: failed to read %s entries: %s", entry.Name(), err.Error())
			return
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

	for _, ref := range refFiles {
		path := filepath.Join(searchDir, ref[:2], ref[2:])

		objType, authorName, authorEmail, timestamp, msg, err :=
			readCommitRef(path)

		if err != nil {
			continue
		}

		respF("%s %s\nAuthor: %s %s\nDate: %s\n\n%s\n\n",
			objType, ref, authorName, authorEmail, timestamp, msg)
	}
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

	if objType != "commit" {
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

func catFile(args []string) {
	run_env := os.Getenv("run_env")

	searchDir := ".git/objects"
	if run_env == "test" {
		searchDir = "tmp/.git/objects"
	}

	if len(args) > 2 {
		inp := args[2]
		if inp == "" {
			respErr("cat-file err: invalid input file use cat-file --list for list of possible hashes to read.")
			return
		}
		objDir := fmt.Sprintf("%s/%s", searchDir, inp[:2])

		stat, err := os.Stat(objDir)
		if err != nil {
			respErrF("cat-file err: the %s reference dir does not exits in %s at %s", inp, searchDir, objDir)
			return
		}
		if !stat.IsDir() {
			respErr("cat-file err: the reference is not a dir")
			return
		}
		entries, err := os.ReadDir(objDir)
		if err != nil {
			respErrF("cat-file err: failed to read %s entries: %s", objDir, err.Error())
			return
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
			respErrF("cat-file err: failed to find any reference with prefix of : %s", inp[2:])
			return
		}
		rp := fmt.Sprintf("%s/%s/%s", searchDir, inp[:2], found)

		_, err = os.Stat(rp)
		if err != nil {
			respErrF("cat-file err: the %s reference does not exits in %s at %s", inp, searchDir, rp)
			return
		}
		refFile, err := os.OpenFile(rp, os.O_RDONLY, 0755)
		if err != nil {
			respErrF("cat-file err: failed to open %s file: %s", rp, err.Error())
			return
		}
		reader, err := zlib.NewReader(refFile)
		if err != nil {
			respErrF("cat-file err: failed to create reader for %s file in zlib: %s", rp, err.Error())
			return
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
			respErrF("cat-file err: failed to read payload of %s object type: %s", err.Error(), objType)
			return
		}
		resp(fmt.Sprintf("%s payload is:\n", rp), string(payload))
		return
	} else {
		respErr("cat-file err: specify which file to read use 'help cat-file' for more info")
		return
	}
}

func initialize() {
	initDir := ".git"
	run_env := os.Getenv("run_env")
	if run_env == "test" {
		initDir = "tmp/.git"
	}
	_, err := os.Stat(initDir)
	if err != nil {
		if err := os.MkdirAll(initDir, 0755); err != nil {
			respErrF("failed to create .git dir: %s", err.Error())
		}
		if err := os.MkdirAll(fmt.Sprintf("%s/objects", initDir), 0755); err != nil {
			respErrF("failed to create %s/objects dir: %s", initDir, err.Error())
		}
		if err := os.MkdirAll(fmt.Sprintf("%s/refs", initDir), 0755); err != nil {
			respErrF("failed to create %s/refs dir: %s", initDir, err.Error())
		}
		if err := os.MkdirAll(fmt.Sprintf("%s/logs", initDir), 0755); err != nil {
			respErrF("failed to create %s/logs dir: %s", initDir, err.Error())
		}
		if err := os.WriteFile(fmt.Sprintf("%s/HEAD", initDir), []byte("ref: refs/heads/main\n"), 0755); err != nil {
			respErrF("failed to write %s/HEAD file: %s", initDir, err.Error())
		}
	} else {
		// we can either recreate .git dir or give error that it exists
		respErr("the .git dir already exists")
	}
}

func help(args []string) {
	if len(args) > 2 {
		subCmd := args[2]
		switch subCmd {
		case "cat-file":
			resp("cat-file <hash>:", "reads the changes of a hash and prints the changes content")
		default:
			respErrF("invalid sub command '%s' use 'help' for list of possible commands", subCmd)
		}
	} else {
		resp("list of possible commands:", "\n\t- help", "\n\t- cat-file")
	}
}

const (
	White = "\033[37m"
	Blue  = "\033[34m"
	Red   = "\033[31m"
	Reset = "\033[0m"
)

// we use our own print methods for later tests to capture stdout and stderr
func respErrF(format string, args ...any) {
	m := new(strings.Builder)
	fmt.Fprintf(m, format, args...) // apply formatting
	m.WriteString("\n")
	os.Stderr.WriteString(m.String())
}
func respErr(msgs ...string) {
	m := new(strings.Builder)
	if len(msgs) > 1 {
		for i, msg := range msgs {
			if i == 0 {
				fmt.Fprintf(m, "%s%s%s ", White, msg, Reset)
			} else {
				fmt.Fprintf(m, "%s%s%s ", Red, msg, Reset)
			}
		}
	} else {
		fmt.Fprintf(m, "%s%s%s ", Red, msgs[0], Reset)
	}

	m.WriteString("\n")
	os.Stderr.WriteString(m.String())
}

func respF(format string, args ...any) {
	m := new(strings.Builder)
	fmt.Fprintf(m, format, args...) // formatting
	m.WriteString("\n")
	os.Stdout.WriteString(m.String())
}
func resp(msgs ...string) {
	m := new(strings.Builder)
	for i, msg := range msgs {
		if i == 0 {
			fmt.Fprintf(m, "%s%s%s", White, msg, Reset)
		} else {
			fmt.Fprintf(m, "%s%s%s", Blue, msg, Reset)
		}
	}
	m.WriteString("\n")
	os.Stdout.WriteString(m.String())
}

package main

import (
	"fmt"
	"os"
	"strings"
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
	case "read-tree":
		if len(args) > 2 {
			inp := args[2]
			if inp == "" {
				respErr("invalid input file use read-tree --list for list of possible hashes to read.")
			}

		} else {
			respErr("specify which file to read use 'help read-tree' for more info")
		}

	default:
		respErrF("invalid command '%s' use help for list of commands", command)
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
		if err := os.MkdirAll(fmt.Sprintf("%s/ref", initDir), 0755); err != nil {
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
		case "read-tree":
			resp("read-tree <hash>:", "reads the changes of a hash and prints the changes content")
		default:
			respErrF("invalid sub command '%s' use 'help' for list of possible commands", subCmd)
		}
	} else {
		resp("list of possible commands:", "\n\t- help", "\n\t- read-tree")
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
			fmt.Fprintf(m, "%s%s%s ", White, msg, Reset)
		} else {
			fmt.Fprintf(m, "%s%s%s ", Blue, msg, Reset)
		}
	}
	m.WriteString("\n")
	os.Stdout.WriteString(m.String())
}

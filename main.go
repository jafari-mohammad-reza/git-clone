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
	for i, msg := range msgs {
		if i == 0 {
			fmt.Fprintf(m, "%s%s%s ", White, msg, Reset)
		} else {
			fmt.Fprintf(m, "%s%s%s ", Red, msg, Reset)
		}
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

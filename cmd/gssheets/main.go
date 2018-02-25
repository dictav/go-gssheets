package main // import "github.com/dictav/go-gssheets/cmd/gssheets"

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dictav/go-gssheets/cmd/gssheets/internal/auth"
	"github.com/dictav/go-gssheets/cmd/gssheets/internal/download"
	"github.com/dictav/go-gssheets/cmd/gssheets/internal/upload"
)

const (
	cmdName = "gssheets"
)

var (
	version = "v0.0.0"
	date    string

	showVersion = flag.Bool("version", false, "show version")
)

var (
	errorWriter = os.Stderr

	commands = []struct {
		Name        string
		Description string
		Run         func([]string) error
		Usage       func()
	}{
		{
			auth.Name,
			auth.Description,
			auth.Run,
			auth.Usage,
		},
		{
			download.Name,
			download.Description,
			download.Run,
			download.Usage,
		},
		{
			upload.Name,
			upload.Description,
			upload.Run,
			upload.Usage,
		},
	}
)

func printVersion() {
	fmt.Fprintf(errorWriter, "%s %s %s\n", cmdName, version, date)
}

//usage prints to stdout information about the tool
func usage() {
	printVersion()
	fmt.Fprintf(errorWriter, "usage: %s <command>\n", strings.ToLower(cmdName))
}

func main() {
	var (
		cmd     func([]string) error
		options []string
	)

	flag.Parse()
	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	options = flag.Args()
	if len(options) == 0 {
		usage()
		os.Exit(1)
	}

	subcmd := options[0]
	options = options[1:]

	for i := range commands {
		if subcmd == commands[i].Name {
			cmd = commands[i].Run
		}
	}

	if cmd == nil {
		usage()
		os.Exit(1)
	}

	if e := cmd(options); e != nil {
		fmt.Fprintf(errorWriter, "%v\n", e)
		os.Exit(9)
	}
}

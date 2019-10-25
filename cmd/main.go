package main

import (
	"os"

	commands "github.com/damoonazarpazhooh/File-Ingestion/cmd/commands"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "Chunker-CLI"
	app.Usage = "File Chunker - merger CLI Demo"
	app.Commands = []cli.Command{
		commands.Sample,
		commands.Splitter,
	}
	app.Run(os.Args)
}

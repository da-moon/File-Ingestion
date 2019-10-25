package commands

import (
	"context"
	"log"
	"path/filepath"

	splitter "github.com/damoonazarpazhooh/chunker"
	"github.com/damoonazarpazhooh/File-Ingestion/internal/uuid"
	utils "github.com/damoonazarpazhooh/File-Ingestion/pkg/utils"
	osext "github.com/kardianos/osext"
	"github.com/urfave/cli"
)

// Splitter ...
var Splitter = cli.Command{
	Name:    "Splitter",
	Aliases: []string{"splitter"},
	Usage:   "split a file into chunks",
	Subcommands: []cli.Command{
		snapshot,
		restore,
	},
}

// singleSampleFile ...
var snapshot = cli.Command{
	Name:    "Snapshot",
	Aliases: []string{"snapshot"},
	Usage:   "takes a snapshot of files in a directory and split them into chunks",
	Description: `this command helps with generating snapshots of a directory and
	converting that snapshot into chunks.
	--tag flag is used to set a tag for the snapshot. if no tag is provided , a
	uuid is used as snapshot tag
	`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "tag",
			Value: "",
			Usage: "tag used to identify this snapshot",
		},
	},
	Action: func(ctx *cli.Context) error {

		path := ctx.Args().First()
		if len(path) == 0 {
			path = "tmp"
			selfPath, _ := osext.ExecutableFolder()
			path = utils.PathJoin(selfPath, path)
		}
		path, _ = filepath.Abs(path)
		filesplitter := splitter.New(
			splitter.LogOps(),
			splitter.WithRootPath(path),
			// splitter.WithChunkSizeInKilobytes(4),
			splitter.WithChunkSizeInMegabytes(4),
			splitter.WithEncryption("encryption-key"),
		)
		tag := ctx.String("tag")
		if len(tag) == 0 {
			tag, _ = uuid.GenerateUUID()
		}
		err := filesplitter.Snapshot(context.Background(), tag)
		if err != nil {
			log.Fatal(err)
		}
		return nil
	},
}

// singleSampleFile ...
var restore = cli.Command{
	Name:    "Restore",
	Aliases: []string{"restore"},
	Usage:   "restores a snapshot from chunks",
	Description: `this command helps with generating restoring snapshots from a directory based on metadatas of a snapshot.
	--tag flag is used to set a tag for the snapshot. if no tag is provided , it would return without any results.
	`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "tag",
			Value: "",
			Usage: "tag used to identify snapshot to restore",
		},
		cli.StringFlag{
			Name:  "restore-root",
			Value: "restore-root-dir",
			Usage: "restore-root is used to pass in the name of the directory in which snapshots are restored",
		},
	},
	Action: func(ctx *cli.Context) error {

		path := ctx.Args().First()
		if len(path) == 0 {
			path = "tmp"
			selfPath, _ := osext.ExecutableFolder()
			path = utils.PathJoin(selfPath, path)
		}
		path, _ = filepath.Abs(path)
		filesplitter := splitter.New(
			splitter.LogOps(),
			splitter.WithRootPath(path),
			// splitter.WithChunkSizeInKilobytes(4),
			splitter.WithChunkSizeInMegabytes(4),
			splitter.WithEncryption("encryption-key"),
		)
		tag := ctx.String("tag")
		if len(tag) == 0 {
			return nil
		}
		restoreRoot := ctx.String("restore-root")

		err := filesplitter.Restore(context.Background(), restoreRoot, tag)
		if err != nil {
			log.Fatal(err)
		}
		return nil
	},
}

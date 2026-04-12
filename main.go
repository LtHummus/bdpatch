package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lthummus/bdpatch/patcher"

	"github.com/urfave/cli/v3"
)

var (
	version    = "dev"
	commit     = "none"
	commitDate = "unknown"
)

const (
	inputPathArgName      = "input path"
	force2DFlagName       = "force-2d"
	noRestructureFlagName = "no-restructure"
)

var cmd = &cli.Command{
	Name:    "bdpatch",
	Usage:   "patch a ripped blu ray disc for Oppo players",
	Version: fmt.Sprintf("bdpatch %s (commit: %s, built: %s)", version, commit, commitDate),
	Action:  run,
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name: inputPathArgName,
		},
	},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  force2DFlagName,
			Usage: "Force a 3D disc to play as 2D (may be required)",
		},
		&cli.BoolFlag{
			Name:  noRestructureFlagName,
			Usage: "Do not restructure the rip in to an AVCHD directory",
		},
	},
}

func run(ctx context.Context, cmd *cli.Command) error {
	inputPath := cmd.StringArg(inputPathArgName)
	if inputPath == "" {
		return fmt.Errorf("input path not set")
	}
	return patcher.PatchDisc(ctx, inputPath, cmd.Bool(force2DFlagName), cmd.Bool(noRestructureFlagName))
}

func main() {
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

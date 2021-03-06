package flags

import (
	"github.com/codegangsta/cli"
)

var (
	DaemonFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Debug log, enabled by default",
		},
		cli.StringFlag{
			Name:  "log",
			Usage: "specific output log file, otherwise output to stdout by default",
		},
		cli.StringFlag{
			Name:  "root",
			Value: "/var/lib/rancher/convoy",
			Usage: "specific root directory of convoy, if configure file exists, daemon specific options would be ignored",
		},
		cli.StringSliceFlag{
			Name:  "drivers",
			Value: &cli.StringSlice{},
			Usage: "Drivers to be enabled, first driver in the list would be treated as default driver",
		},
		cli.StringSliceFlag{
			Name:  "driver-opts",
			Value: &cli.StringSlice{},
			Usage: "options for driver",
		},
		cli.StringFlag{
			Name:  "mnt-ns",
			Usage: "Specify mount namespace file descriptor if user don't want to mount in current namespace. Support by Device Mapper and EBS",
		},
		cli.BoolFlag{
			Name:  "ignore-docker-delete",
			Usage: "Do not delete volumes when told to by Docker",
		},
	}
)

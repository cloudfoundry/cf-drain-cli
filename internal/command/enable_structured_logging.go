package command

import (
	"code.cloudfoundry.org/cli/plugin"
	"github.com/jessevdk/go-flags"
)

type enableStructuredLoggingArgs struct {
	AppName               string
	DrainName             string `long:"drain-name"`
	StructuredLoggingType string
}

func EnableStructuredLogging(
	cli plugin.CliConnection,
	args []string,
	d Downloader,
	log Logger,
) {
	opts := enableStructuredLoggingArgs{}

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	args, err := parser.ParseArgs(args)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if len(args) != 2 {
		log.Fatalf("Invalid arguments, expected 2, got %d.", len(args))
	}

	opts.AppName = args[0]
	opts.StructuredLoggingType = args[1]

	CreateDrain(
		cli,
		[]string{
			opts.AppName,
			"prism://" + opts.StructuredLoggingType,
			"--drain-name",
			opts.DrainName,
		},
		d,
		log,
	)
}

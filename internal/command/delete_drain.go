package command

import (
	"io"

	"code.cloudfoundry.org/cli/plugin"
)

func DeleteDrain(cli plugin.CliConnection, args []string, log Logger, in io.Reader, serviceDrainDrainRemover DrainRemover, appDrainDrainRemover DrainRemover) {
	if len(args) != 1 {
		log.Fatalf("Invalid arguments, expected 1, got %d.", len(args))
	}

	drainName := args[0]

	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}

	var ok bool
	if ok, err = serviceDrainDrainRemover.DeleteDrainAndUser(space.Guid, drainName); ok {
		return
	}
	if err != nil {
		log.Fatalf("%s", err)
	}

	if ok, err = appDrainDrainRemover.DeleteDrainAndUser(space.Guid, drainName); ok {
		return
	}
	if err != nil {
		log.Fatalf("%s", err)
	}

}

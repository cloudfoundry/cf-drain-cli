package command

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"text/tabwriter"

	"code.cloudfoundry.org/cf-drain-cli/internal/drain"
	"code.cloudfoundry.org/cli/plugin"
)

type DrainFetcher interface {
	Drains(spaceGUID string) ([]drain.Drain, error)
}

func Drains(
	cli plugin.CliConnection,
	args []string,
	log Logger,
	tableWriter io.Writer,
	fetchers ...DrainFetcher,
) {
	if len(args) != 0 {
		log.Fatalf("Invalid arguments, expected 0, got %d.", len(args))
	}

	space, err := cli.GetCurrentSpace()
	if err != nil {
		log.Fatalf("%s", err)
	}

	var drains []drain.Drain

	for _, f := range fetchers {
		d, err := f.Drains(space.Guid)
		if err != nil {
			log.Fatalf("Failed to fetch drains: %s", err)
		}
		drains = append(drains, d...)
	}

	if err != nil {
		log.Fatalf("Failed to fetch drains: %s", err)
	}
	tw := tabwriter.NewWriter(tableWriter, 10, 2, 2, ' ', 0)

	// Header
	fmt.Fprintln(tw, "App\tDrain\tType\tURL")
	for _, d := range drains {
		for _, app := range d.Apps {
			drain := []string{
				app,
				d.Name,
				strings.Title(d.Type),
				sanitizeDrainURL(d.DrainURL),
			}
			fmt.Fprintln(tw, strings.Join(drain, "\t"))
		}
	}

	tw.Flush()
}

func sanitizeDrainURL(drainURL string) string {
	u, err := url.Parse(drainURL)
	if err != nil {
		return "failed to parse drain URL"
	}

	if u.User != nil {
		u.User = url.UserPassword("---REDACTED---", "---REDACTED---")
	}

	query := u.Query()
	delete(query, "drain-type")

	for k, v := range query {
		for i := range v {
			query[k][i] = "---REDACTED---"
		}
	}
	u.RawQuery = query.Encode()

	return strings.Replace(u.String(), "---REDACTED---", "<redacted>", -1)
}

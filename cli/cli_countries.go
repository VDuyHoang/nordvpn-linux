package cli

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/NordSecurity/nordvpn-linux/daemon/pb"
	"github.com/NordSecurity/nordvpn-linux/internal"

	"github.com/urfave/cli/v2"
)

// CountriesUsageText is shown next to countries command by nordvpn --help
const CountriesUsageText = "Shows a list of countries where servers are available"

func (c *cmd) Countries(ctx *cli.Context) error {
	resp, err := c.client.Countries(context.Background(), &pb.CountriesRequest{
		Protocol:  c.config.Protocol,
		Obfuscate: c.config.Obfuscate,
	})
	if err != nil {
		return formatError(err)
	}

	countryList, err := internal.Columns(resp.Data)
	if err != nil {
		log.Println(internal.ErrorPrefix, err)
		fmt.Println(strings.Join(resp.Data, ", "))
	} else {
		fmt.Println(countryList)
	}
	return nil
}

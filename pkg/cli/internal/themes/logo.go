package themes

import (
	_ "embed"

	"github.com/fatih/color"
)

//go:embed kmcp-ascii.txt
var kmcpLogoASCII string

func KmcpLogo() string {
	return kmcpLogoASCII
}

func ColoredKmcpLogo() string {
	return ColorPrimary().Add(color.Bold).Sprint(KmcpLogo())
}

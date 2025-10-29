package themes

import (
	_ "embed"

	"github.com/fatih/color"
)

//go:embed kmcp-ascii.txt
var kmcpLogoAscii string

func KmcpLogo() string {
	return kmcpLogoAscii
}

func ColoredKmcpLogo() string {
	return ColorPrimary().Add(color.Bold).Sprint(KmcpLogo())
}

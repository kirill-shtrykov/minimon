package flags

import (
	"flag"
	"os"
	"strings"
)

const (
	addrHelpText = `
The address to listen.
Overrides the MINIMON_ADDR environment variable if set.
Default = :6012
`
	confHelpText = `
Config file path.
Overrides the MINIMON_CONF environment variable if set.
Default = /etc/minimon/config.yaml
`
)

type Flags struct {
	Addr  string
	Conf  string
	Debug bool
}

// Retrieves the value of the environment variable named by the `key`.
// It returns the value if variable present and value not empty.
// Otherwise it returns string value `def`.
func stringFromEnv(key string, def string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimSpace(v)
	}

	return def
}

func Parse() Flags {
	flags := Flags{
		Addr: stringFromEnv("MINIMON_ADDR", ":6012"),
		Conf: stringFromEnv("MINIMON_CONF", "/etc/minimon/config.yaml"),
	}

	flag.StringVar(&flags.Addr, "address", flags.Addr, strings.TrimSpace(addrHelpText))
	flag.StringVar(&flags.Conf, "config", flags.Conf, strings.TrimSpace(confHelpText))
	flag.BoolVar(&flags.Debug, "debug", false, "Enables debug mode")
	flag.Parse()

	return flags
}

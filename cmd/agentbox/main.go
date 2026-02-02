package main

import (
	"os"

	"github.com/davidsenack/agentbox/internal/cmd"
)

// Build-time variables (set via ldflags)
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	cmd.SetVersion(version, commit, buildTime)
	os.Exit(cmd.Execute())
}

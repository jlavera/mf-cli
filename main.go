package main

import "github.com/jlavera/mf-cli/cmd"

// version is set at build time via ldflags: -X main.version=v1.2.3
var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}

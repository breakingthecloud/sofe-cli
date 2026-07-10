package main

import "github.com/breakingthecloud/sofe-cli/cmd"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Version = version
	cmd.Commit = commit
	cmd.BuildDate = date
	cmd.Execute()
}

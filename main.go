package main

import "github.com/breakingthecloud/sofe-cli/cmd"

var version = "dev"

func main() {
	cmd.Version = version
	cmd.Execute()
}

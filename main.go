// Package main causes plugins to be loaded and compiled in.
package main

import (
	"github.com/zostay/garotate/cmd"
	_ "github.com/zostay/garotate/pkg/plugin/circleci"
	_ "github.com/zostay/garotate/pkg/plugin/github"
	_ "github.com/zostay/garotate/pkg/plugin/iam"
)

// main executes the command.
func main() {
	cmd.Execute()
}

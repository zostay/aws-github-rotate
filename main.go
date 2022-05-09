// Package main causes plugins to be loaded and compiled in.
package main

import (
	"github.com/zostay/garotate/cmd"
	_ "github.com/zostay/garotate/pkg/plugin/aws/iam/user/access"
	_ "github.com/zostay/garotate/pkg/plugin/circleci/project/env"
	_ "github.com/zostay/garotate/pkg/plugin/github/action/secret"
)

// main executes the command.
func main() {
	cmd.Execute()
}

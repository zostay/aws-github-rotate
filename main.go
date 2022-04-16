// Package main causes plugins to be loaded and compiled in.
package main

import (
	"github.com/zostay/aws-github-rotate/cmd"
	_ "github.com/zostay/aws-github-rotate/pkg/plugin/github"
	_ "github.com/zostay/aws-github-rotate/pkg/plugin/iam"
)

// main executes the command.
func main() {
	cmd.Execute()
}

package main

import (
	"github.com/zostay/aws-github-rotate/cmd"
	_ "github.com/zostay/aws-github-rotate/pkg/plugin/github"
	_ "github.com/zostay/aws-github-rotate/pkg/plugin/iam"
)

func main() {
	cmd.Execute()
}

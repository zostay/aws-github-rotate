package iam

import (
	"context"
	"reflect"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/zostay/aws-github-rotate/pkg/config"
	"github.com/zostay/aws-github-rotate/pkg/plugin"
)

// builder implements the plugin.Builder interface and provides a factory method
// for constructing an IAM client.
type builder struct{}

// Build constructs and returns an IAM client.
func (b *builder) Build(ctx context.Context, c *config.Plugin) (plugin.Instance, error) {
	session := session.Must(session.NewSession())
	svcIam := iam.New(session)

	return &Client{svcIam}, nil
}

// init registers the plugin.
func init() {
	pkg := reflect.TypeOf(Client{}).PkgPath()
	plugin.Register(pkg, new(builder))
}

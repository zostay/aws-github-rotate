package env

import (
	"context"
	"net/http"
	"os"
	"reflect"

	"github.com/zostay/garotate/pkg/config"
	"github.com/zostay/garotate/pkg/plugin"
)

// builder implements the plugin.Builder interface and provides the factory
// method for constructing a Client.
type builder struct{}

// defaults provides a sane default configuration for CircleCI.
var defaultHost = "https://circleci.com"
var defaultRestEndpoint = "/api/v2"

// Build constructs and returns a CircleCI client.
func (b *builder) Build(
	ctx context.Context,
	c *config.Plugin,
) (plugin.Instance, error) {
	hc := http.DefaultClient
	token := os.Getenv("CIRCLECI_TOKEN")
	return &Client{hc, token}, nil
}

// init registers the plugin.
func init() {
	pkg := reflect.TypeOf(Client{}).PkgPath()
	plugin.Register(pkg, new(builder))
}

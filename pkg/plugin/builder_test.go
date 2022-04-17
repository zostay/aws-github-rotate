package plugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zostay/garotate/pkg/config"
)

type errorPlugin struct{}

func (*errorPlugin) Build(ctx context.Context, c *config.Plugin) (Instance, error) {
	return nil, fmt.Errorf("bad stuff")
}

type nopPlugin struct{}
type nopInstance struct{}

func (*nopPlugin) Build(ctx context.Context, c *config.Plugin) (Instance, error) {
	return new(nopInstance), nil
}

func (*nopInstance) Name() string {
	return "nop"
}

func init() {
	Register(
		"github.com/zostay/garotate/pkg/plugin/builder_test/error",
		new(errorPlugin),
	)
	Register(
		"github.com/zostay/garotate/pkg/plugin/builder_test/nop",
		new(nopPlugin),
	)
}

func TestSadMissingPlugin(t *testing.T) {
	m := NewManager(
		config.PluginList{},
	)
	require.NotNil(t, m, "got a manager")

	ctx := context.Background()
	inst, err := m.Instance(ctx, "nope")
	assert.Nil(t, inst, "non-existent name gets no plugin")
	assert.ErrorContains(t, err, "no plugin configuration found",
		"error with bad plugin name")
}

func TestSadPluginError(t *testing.T) {
	m := NewManager(
		config.PluginList{
			"error": config.Plugin{
				Package: "github.com/zostay/garotate/pkg/plugin/builder_test/error",
			},
		},
	)
	require.NotNil(t, m, "got a manager")

	ctx := context.Background()
	inst, err := m.Instance(ctx, "error")
	assert.Nil(t, inst, "plugin that errors out gets no plugin")
	assert.ErrorContains(t, err, "error while building",
		"error with broken plugin")
	assert.ErrorContains(t, err, "bad stuff",
		"error contains plugin error")
}

func TestHappyPlugin(t *testing.T) {
	m := NewManager(
		config.PluginList{
			"nop": config.Plugin{
				Package: "github.com/zostay/garotate/pkg/plugin/builder_test/nop",
			},
		},
	)
	require.NotNil(t, m, "got a manager")

	ctx := context.Background()
	inst, err := m.Instance(ctx, "nop")
	assert.NotNil(t, inst, "plugin works")
	assert.NoError(t, err, "no error")
	assert.Equal(t, inst.Name(), "nop", "got the expected plugin")

	inst2, err2 := m.Instance(ctx, "nop")
	assert.NotNil(t, inst2, "plugin works a second time")
	assert.NoError(t, err2, "no error a second time")
	assert.Equal(t, inst2.Name(), "nop", "got the expected plugin the second time")

	assert.Same(t, inst, inst2, "second retrieval was the cached value")
}

func TestSadBuildFuncMissingPlugin(t *testing.T) {
	ctx := context.Background()
	inst, err := Build(ctx, &config.Plugin{
		Name:    "foo",
		Package: "github.com/zostay/garotate/pkg/plugin/builder_test/nope",
	})
	assert.Nil(t, inst, "non-existent name gets no plugin")
	assert.ErrorContains(t, err, "no plugin found for package",
		"error with bad plugin name")
}

func TestSadPanicOnReRegister(t *testing.T) {
	assert.Panics(t,
		func() {
			Register(
				"github.com/zostay/garotate/pkg/plugin/builder_test/nop",
				new(nopPlugin),
			)
		},
		"registering a plugin again results in panic",
	)
}

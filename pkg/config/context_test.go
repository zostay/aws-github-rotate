package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHappyDefaultLogger(t *testing.T) {
	assert.NotNil(t, DefaultLogger, "default logger is set at start")
}

func TestHappyProductionLogger(t *testing.T) {
	pl := ProductionLogger()
	assert.NotNil(t, pl, "production logger returns a value")
}

func TestHappyDevelopmentLogger(t *testing.T) {
	dl := DevelopmentLogger()
	assert.NotNil(t, dl, "development logger returns a value")
}

func TestHappyContextLogger(t *testing.T) {
	l := DefaultLogger()
	assert.NotNil(t, l, "default logger produces a value")

	ctx := context.Background()
	assert.NotNil(t, ctx, "we actually have a context")

	ctx = WithLogger(ctx, l)
	assert.NotNil(t, ctx, "WithLogger returns a context")

	l2 := LoggerFrom(ctx)
	assert.NotNil(t, l2, "LoggerFrom returns a logger")
	assert.Same(t, l, l2, "LoggerFrom returns the same logger as put in WithLogger")

	ctx = context.Background()
	assert.NotNil(t, ctx, "we actually have a fresh context again")
	l3 := LoggerFrom(ctx)
	assert.NotNil(t, l3, "LoggerFrom returns a logger even without one set")
	assert.NotSame(t, l, l3, "LoggerFrom made a new one from scratch to do it")
}

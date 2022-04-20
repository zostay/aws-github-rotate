package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	err := NewAggregate([]error{
		fmt.Errorf("one"),
		fmt.Errorf("two"),
	})
	require.NotNil(t, err, "NewAggregate returns a value")
	assert.Error(t, err, "NewAggregate returns an error")
	assert.Equal(t, err.Error(), "one; two", "error mesasge is as expected")
	assert.Equal(t, err.Errors(), []error{
		fmt.Errorf("one"),
		fmt.Errorf("two"),
	}, "original error list is recoverable")
}

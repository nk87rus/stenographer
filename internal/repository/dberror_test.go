package repository

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	var s = "test"
	require.Equal(t, s, (&DBError{Err: errors.New(s)}).Error())
}

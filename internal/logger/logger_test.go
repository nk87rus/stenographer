package logger

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Init()
	log.Info().Msg("Hello")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	require.Contains(t, buf.String(), "INFO")
	require.Contains(t, buf.String(), "Hello")
}

func TestFormatLevel(t *testing.T) {
	resultData := formatLevel("debug")
	require.Equal(t, "DEBUG ", resultData)
}

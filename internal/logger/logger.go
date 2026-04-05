package logger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const logLevel = zerolog.DebugLevel

func Init() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	output.FormatLevel = formatLevel
	log.Logger = zerolog.New(output).With().Timestamp().Logger().Level(logLevel)
}

func formatLevel(i any) string {
	return strings.ToUpper(fmt.Sprintf("%-6s", i))
}

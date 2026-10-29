package logger

import (
	"os"

	"github.com/aoamusat/hexgogen/domain/services"
	"github.com/rs/zerolog"
)

type zerologLog struct {
	client zerolog.Logger
}

var _ services.Logger = (*zerologLog)(nil)

// NewZerolog creates a timestamped JSON logger that writes to stdout.
func NewZerolog() *zerologLog {
	return &zerologLog{client: zerolog.New(os.Stdout).With().Timestamp().Logger()}
}

func (log *zerologLog) Info(msg string)  { log.client.Info().Msg(msg) }
func (log *zerologLog) Debug(msg string) { log.client.Debug().Msg(msg) }
func (log *zerologLog) Warn(msg string)  { log.client.Warn().Msg(msg) }
func (log *zerologLog) Error(msg string, err error) {
	log.client.Error().Err(err).Msg(msg)
}

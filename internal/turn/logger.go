package turn

import (
	"github.com/pion/logging"
	"github.com/rs/zerolog"
)

type pionLogger struct {
	*zerolog.Logger
}

func adapter(logger *pionLogger) logging.LoggerFactory {
	return logger
}

func (l *pionLogger) NewLogger(scope string) logging.LeveledLogger {
	return l
}

func (l *pionLogger) Trace(msg string) {
	l.Logger.Trace().Msg(msg)
}

func (l *pionLogger) Tracef(format string, args ...interface{}) {
	l.Logger.Trace().Msgf(format, args...)
}

func (l *pionLogger) Debug(msg string) {
	l.Logger.Debug().Msg(msg)
}

func (l *pionLogger) Debugf(format string, args ...interface{}) {
	l.Logger.Debug().Msgf(format, args...)
}

func (l *pionLogger) Info(msg string) {
	l.Logger.Info().Msg(msg)
}

func (l *pionLogger) Infof(format string, args ...interface{}) {
	l.Logger.Info().Msgf(format, args...)
}

func (l *pionLogger) Warn(msg string) {
	l.Logger.Warn().Msg(msg)
}

func (l *pionLogger) Warnf(format string, args ...interface{}) {
	l.Logger.Warn().Msgf(format, args...)
}

func (l *pionLogger) Error(msg string) {
	l.Logger.Error().Msg(msg)
}

func (l *pionLogger) Errorf(format string, args ...interface{}) {
	l.Logger.Error().Msgf(format, args...)
}

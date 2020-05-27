package loadgen

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type correlationIdType int

const (
	requestIdKey correlationIdType = iota
	sessionIdKey
)

type Logger struct {
	*zap.SugaredLogger
}

var logger *Logger

// WithRqId returns a context which knows its request ID
func WithRqId(ctx context.Context, rqId string) context.Context {
	return context.WithValue(ctx, requestIdKey, rqId)
}

// WithSessionId returns a context which knows its session ID
func WithSessionId(ctx context.Context, sessionId string) context.Context {
	return context.WithValue(ctx, sessionIdKey, sessionId)
}

// Logger returns a zap logger with as much context as possible
func (m *Logger) FromCtx(ctx context.Context) *Logger {
	newLogger := logger
	if ctx != nil {
		if ctxRqId, ok := ctx.Value(requestIdKey).(string); ok {
			newLogger = &Logger{newLogger.With(zap.String("rqId", ctxRqId))}
		}
		if ctxSessionId, ok := ctx.Value(sessionIdKey).(string); ok {
			newLogger = &Logger{newLogger.With(zap.String("sessionId", ctxSessionId))}
		}
	}
	return newLogger
}

func setupLogger(encoding string, level string) *Logger {
	rawJSON := []byte(fmt.Sprintf(`{
	  "level": "%s",
	  "encoding": "%s",
	  "outputPaths": ["stdout", "/tmp/logs"],
	  "errorOutputPaths": ["stderr"],
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
		"levelEncoder": "uppercase",
        "timeKey": "time",
		"timeEncoder": "ISO8601",
		"callerKey": "caller",
		"callerEncoder": "short"
	  }
	}`, level, encoding))

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	log = &Logger{logger.Sugar()}
	return log
}

func NewLogger() *Logger {
	lvl := viper.GetString("logging.level")
	encoding := viper.GetString("logging.encoding")
	return setupLogger(encoding, lvl)
}

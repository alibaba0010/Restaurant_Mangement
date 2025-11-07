package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

// customTimeEncoder serializes a time.Time to a human-friendly format
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

func InitLogger() {
	// Configure console output with colors
	config := zap.NewDevelopmentEncoderConfig()
	config.EncodeLevel = zapcore.CapitalColorLevelEncoder // Enable colors
	config.EncodeTime = customTimeEncoder                 // Human-friendly timestamp
	config.EncodeCaller = nil                            // Disable caller
	config.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.ConsoleSeparator = " "                        // Clean spacing between fields
	
	// Create console encoder
	consoleEncoder := zapcore.NewConsoleEncoder(config)
	
	// Write to stderr for console output
	core := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stderr),
		zapcore.InfoLevel,
	)

	// Create logger without caller or stacktrace
	Log = zap.New(core)
}

func Sync(){
	if Log != nil {
		_ = Log.Sync()
	}
}
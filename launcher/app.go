package launcher

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type AppDef struct {
	ID            string
	Title         string
	Description   string
	MetricsID     string
	Logger        *LoggingDef
	RegisterFlags func(cmd *cobra.Command) error
	InitFunc      func(runtime *Runtime) error
	FactoryFunc   func(runtime *Runtime) (App, error)
}

type LoggingDef struct {
	Levels []zapcore.Level
	Regex  string
}

func NewLoggingDef(regex string, levels []zapcore.Level) *LoggingDef {
	if len(levels) == 0 {
		levels = []zapcore.Level{zap.WarnLevel, zap.InfoLevel, zap.InfoLevel, zap.DebugLevel}
	}

	return &LoggingDef{
		Levels: levels,
		Regex:  regex,
	}
}

type App interface {
	Terminating() <-chan struct{}
	Terminated() <-chan struct{}
	Shutdown(err error)
	Err() error
	Run() error
}

type readiable interface {
	IsReady() bool
}

//go:generate go-enum -f=$GOFILE --marshal --names

//
// ENUM(
//   NotFound
//   Created
//   Running
//   Warning
//   Stopped
// )
//
type AppStatus uint

type AppInfo struct {
	ID     string
	Status AppStatus
}

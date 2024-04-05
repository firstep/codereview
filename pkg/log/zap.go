package log

import (
	"os"
	"time"

	"github.com/firstep/aries/config"
	logger "github.com/firstep/aries/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ZapLogger struct {
	logger      *zap.Logger
	atomicLevel zap.AtomicLevel
}

func init() {
	level := config.GetString("log.level", "info")
	path := config.GetString("log.path", "./codeview.log")
	logger.SetLogger(NewZapLogger(level, path))
}

func NewZapLogger(defaultLevel string, logPath string) *ZapLogger {
	atomicLevel, err := zap.ParseAtomicLevel(defaultLevel)
	if err != nil {
		panic("error log level," + err.Error())
	}

	coreList := make([]zapcore.Core, 0, 2)

	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
	cfg.ConsoleSeparator = " "
	encoder := zapcore.NewConsoleEncoder(cfg)
	coreList = append(coreList, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.InfoLevel))

	fileLogger := lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    config.GetInt("log.maxSize", 100), //megabytes
		MaxBackups: config.GetInt("log.maxBackup", 3),
		MaxAge:     config.GetInt("log.maxAge", 30), //days
		LocalTime:  true,
		Compress:   false,
	}
	syncWriter := &zapcore.BufferedWriteSyncer{
		WS:   zapcore.AddSync(&fileLogger),
		Size: config.GetInt("log.buffer", 1024),
	}
	coreList = append(coreList, zapcore.NewCore(encoder, syncWriter, atomicLevel))

	core := zapcore.NewTee(coreList...)

	logger := zap.New(core, zap.AddStacktrace(zap.ErrorLevel))

	return &ZapLogger{logger: logger, atomicLevel: atomicLevel}
}

func (z *ZapLogger) Debug(args ...any) {
	z.logger.Sugar().Debugln(args...)
}

func (z *ZapLogger) Debugf(template string, args ...any) {
	z.logger.Sugar().Debugf(template, args...)
}

func (z *ZapLogger) Debugw(template string, args ...any) {
	z.logger.Sugar().Debugw(template, args...)
}

func (z *ZapLogger) IsDebugEnabled() bool {
	return z.atomicLevel.Enabled(zap.DebugLevel)
}

func (z *ZapLogger) Info(args ...any) {
	z.logger.Sugar().Infoln(args...)
}

func (z *ZapLogger) Infof(template string, args ...any) {
	z.logger.Sugar().Infof(template, args...)
}

func (z *ZapLogger) Infow(template string, args ...any) {
	z.logger.Sugar().Infow(template, args...)
}

func (z *ZapLogger) IsInfoEnabled() bool {
	return z.atomicLevel.Enabled(zap.InfoLevel)
}

func (z *ZapLogger) Warn(args ...any) {
	z.logger.Sugar().Warnln(args...)
}

func (z *ZapLogger) Warnf(template string, args ...any) {
	z.logger.Sugar().Warnf(template, args...)
}

func (z *ZapLogger) Warnw(template string, args ...any) {
	z.logger.Sugar().Warnw(template, args...)
}

func (z *ZapLogger) IsWarnEnabled() bool {
	return z.atomicLevel.Enabled(zap.WarnLevel)
}

func (z *ZapLogger) Error(args ...any) {
	z.logger.Sugar().Errorln(args...)
}

func (z *ZapLogger) Errorf(template string, args ...any) {
	z.logger.Sugar().Errorf(template, args...)
}

func (z *ZapLogger) Errorw(template string, args ...any) {
	z.logger.Sugar().Errorw(template, args...)
}

func (z *ZapLogger) IsErrorEnabled() bool {
	return z.atomicLevel.Enabled(zap.ErrorLevel)
}

func (z *ZapLogger) SetLevel(levelStr string) error {
	level, err := zapcore.ParseLevel(levelStr)
	if err != nil {
		return err
	}

	z.atomicLevel.SetLevel(level)

	return nil
}

func (z *ZapLogger) Flush() {
	z.logger.Sync()
}

package qiniu

import (
	"log"
	"os"
)

// LogLevelType 定义了日志输出的级别， 用来指导哪些日志可以输出
type LogLevelType uint

// Value 返回日志输出级别， 如果l的值为nil， 那么返回LogOff
func (l *LogLevelType) Value() LogLevelType {
	if l != nil {
		return *l
	}
	return LogOff
}

// Matches 返回true如果日志级别v被开启，可以输出日志
func (l *LogLevelType) Matches(v LogLevelType) bool {
	c := l.Value()
	return c&v == v
}

// AtLeast 返回true,  如果l的级别大于等于v， 否则返回false
func (l *LogLevelType) AtLeast(v LogLevelType) bool {
	c := l.Value()
	return c >= v
}

const (
	// LogOff 关闭所有的日志输出，这个是SDK默认的状态
	LogOff LogLevelType = iota * 0x1000

	// LogDebug 用来给SDK调试输出日志
	LogDebug
)

const (
	// LogDebugWithHTTPBody 输出请求和响应的头信息， 体信息
	LogDebugWithHTTPBody LogLevelType = LogDebug | (1 << iota)

	// LogDebugWithRequestRetries 当请求重试的时候，输出日志
	LogDebugWithRequestRetries

	// LogDebugWithRequestErrors 开启日志输出，当请求在build, send, validate, unmarshal阶段失败的时候
	LogDebugWithRequestErrors

	// LogDebugMultipartUpload 开启分片上传调试日志
	LogDebugMultipartUpload
)

// Logger 是最小化的日志输出接口
type Logger interface {
	Log(...interface{})
}

// LoggerFunc 用来封装函数，方便地实现Logger接口
type LoggerFunc func(...interface{})

// Log 用参数args, 调用封装的函数
func (f LoggerFunc) Log(args ...interface{}) {
	f(args...)
}

// NewDefaultLogger 返回一个Logger对象
func NewDefaultLogger() Logger {
	return &defaultLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

type defaultLogger struct {
	logger *log.Logger
}

// Log logs the parameters to the stdlib logger. See log.Println.
func (l defaultLogger) Log(args ...interface{}) {
	l.logger.Println(args...)
}

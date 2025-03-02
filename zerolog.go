package zerolog

import (
	"os"
	"regexp"
	"strconv"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/yeencloud/logger"
)

type ZeroLogMiddleware struct {
	json bool
}

// MARK: - Colorization

const clearColor = "\x1b[0m"
const greenColor = "\x1b[32m"
const redColor = "\x1b[31m"
const yellowColor = "\x1b[33m"
const blueColor = "\x1b[34m"
const magentaColor = "\x1b[35m"
const cyanColor = "\x1b[36m"
const lightMagentaColor = "\x1b[95m"

func (z *ZeroLogMiddleware) colorForMethod(method string) string {
	switch method {
	case "GET":
		return greenColor
	case "POST":
		return blueColor
	case "PUT":
		return cyanColor
	case "DELETE":
		return redColor
	case "PATCH":
		return yellowColor
	default:
		return clearColor
	}
}

func (z *ZeroLogMiddleware) colorForStatus(status int) string {
	// So gocritic wants me to use a switch statement here instead of if/else but how the f am I supposed to do that with ranges?
	//nolint:all
	if status >= 200 && status < 300 {
		return greenColor
	} else if status >= 300 && status < 400 {
		return yellowColor
	} else if status >= 400 && status < 500 {
		return redColor
	} else if status >= 500 {
		return lightMagentaColor
	}

	return clearColor
}

func (z *ZeroLogMiddleware) colorRestMethodPath(str string) string {
	regex := regexp.MustCompile(`^(?P<method>\w+)\s(?P<path>/\S+)$`)

	if regex.MatchString(str) {
		method := regex.FindStringSubmatch(str)[1]
		path := regex.FindStringSubmatch(str)[2]
		color := z.colorForMethod(method)

		return color + method + clearColor + " " + path
	}

	return str
}

func (z *ZeroLogMiddleware) colorRestMethodStatus(str string) string {
	regex := regexp.MustCompile(`^(?P<method>\w+)\s(?P<path>/\S*)\s(?P<status>\d+)\s(?P<message>.+)$`)

	if regex.MatchString(str) {
		method := regex.FindStringSubmatch(str)[1]
		path := regex.FindStringSubmatch(str)[2]
		status := regex.FindStringSubmatch(str)[3]
		message := regex.FindStringSubmatch(str)[4]

		statusCode, _ := strconv.Atoi(status)

		methodColor := z.colorForMethod(method)
		statusColor := z.colorForStatus(statusCode)

		return methodColor + method + clearColor + " " + path + " " + statusColor + status + clearColor + " - " + message
	}

	return str
}

func (z *ZeroLogMiddleware) colorize(str string) string {
	if z.json {
		return str
	}
	str = z.colorRestMethodPath(str)
	str = z.colorRestMethodStatus(str)

	return str
}

// MARK: - Logging

func (z *ZeroLogMiddleware) Log(message Logger.StandardMessage) {
	level := zerolog.NoLevel

	switch message.Level {
	case LoggerDomain.LogLevelDebug:
		level = zerolog.DebugLevel
	case LoggerDomain.LogLevelInfo:
		level = zerolog.InfoLevel
	case LoggerDomain.LogLevelWarn:
		level = zerolog.WarnLevel
	case LoggerDomain.LogLevelError:
		level = zerolog.ErrorLevel
	case LoggerDomain.LogLevelFatal:
		level = zerolog.FatalLevel
	case LoggerDomain.LogLevelPanic:
		level = zerolog.PanicLevel
	case LoggerDomain.LogLevelSQL:
		level = zerolog.TraceLevel
	}

	currentLog := zlog.WithLevel(level)

	for field, value := range message.Fields {
		// as this middleware is used for terminal logs principally for the dev environment, we don't want to pollute the logs with the trace dump
		if field == LoggerDomain.LogFieldTraceDump {
			continue
		}

		err, valid := value.(error)
		if field == LoggerDomain.LogFieldError && valid {
			currentLog = currentLog.Any(field.String(), err.Error())
		} else {
			currentLog = currentLog.Any(field.String(), value)
		}
	}

	msgStr := message.Message

	if message.Level == LoggerDomain.LogLevelSQL {
		msgStr = magentaColor + msgStr + clearColor
	}
	currentLog.Msg(z.colorize(msgStr))
}

func NewZeroLogMiddleware(outputJson bool) *ZeroLogMiddleware {
	if outputJson {
		zlog.Logger = zlog.Output(os.Stderr)
	} else {
		zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	return &ZeroLogMiddleware{
		json: outputJson,
	}
}

package logger

import (
	"fmt"
	"net/http"
)

type Log struct {
	Writer http.ResponseWriter
}

func (l *Log) Write(format string, a ...interface{}) {
	l.write(fmt.Sprintf(format, a...) + "<br>")
}

func (l *Log) write(str string) {
	l.Writer.Write([]byte(str))
	l.Writer.(http.Flusher).Flush()
}

func CreateLog(w http.ResponseWriter) Log {
	log := Log{
		Writer: w,
	}
	log.write(`
    <style>
        body {
            font-family: monospace;
            padding: 10px;
            color: white;
            background-color: #333;
        }
    </style>
    `)
	return log
}

package logger

import (
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

var Log = log.New()

func InitializeLogger(format log.Formatter, level log.Level, output io.Writer) *log.Logger {
	Log.SetFormatter(format)
	Log.SetOutput(output)
	Log.SetLevel(level)
	return Log
}

type Fields log.Fields

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func ResponceLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responce http.ResponseWriter, request *http.Request) {
		start := time.Now()
		responseData := responseData{
			status: 0,
			size:   0,
		}
		newResp := loggingResponseWriter{
			ResponseWriter: responce,
			responseData:   &responseData,
		}

		next.ServeHTTP(&newResp, request)

		Log.WithFields(log.Fields{
			"URL":            request.URL,
			"method":         request.Method,
			"execution time": time.Since(start),
			"size":           newResp.responseData.size,
			"status":         newResp.responseData.status,
			"cookie":         request.Cookies(),
		}).Info("User request")
	})
}

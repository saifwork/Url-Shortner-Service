package middleware

import (
	"bytes"
	"io"
	"log"
	"os"

	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/saifwork/url-shortner-service/app/configs"
)

var LogFile *os.File

// DefaultStructuredLogger logs a gin HTTP request in JSON format. Uses the
// default logger from rs/zerolog.
func DefaultStructuredLogger(conf *configs.Config, logfile *os.File) gin.HandlerFunc {
	LogFile = logfile
	logger := zerolog.New(LogFile).With().Timestamp().Logger()
	return StructuredLogger(conf, &logger)
}

// StructuredLogger logs a gin HTTP request in JSON format. Allows to set the
// logger for testing purposes.
func StructuredLogger(conf *configs.Config, logger *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Do not log any swagger requests
		if strings.Contains(c.FullPath(), "swagger") {
			c.Next()
			return
		}

		if _, err := os.Stat(GetLogfilePath(conf)); os.IsNotExist(err) {
			LogFile, _ = os.OpenFile(GetLogfilePath(conf), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
			logg := zerolog.New(LogFile).With().Timestamp().Logger()
			logger = &logg
		}

		start := time.Now() // Start timer
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		buf, _ := io.ReadAll(c.Request.Body)
		rdr1 := io.NopCloser(bytes.NewBuffer(buf))
		rdr2 := io.NopCloser(bytes.NewBuffer(buf)) //We have to create a new Buffer, because rdr1 will be read

		space := regexp.MustCompile(`\s+`)
		reqBody := readBody(rdr1)
		reqBody = strings.Replace(reqBody, "\n", "", -1)
		// Remove double spaces from the string
		reqBody = space.ReplaceAllString(reqBody, " ")

		c.Request.Body = rdr2
		w := &responseBodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = w

		// Process request
		//c.Request.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
		c.Next()

		// Fill the params
		go func() {
			param := gin.LogFormatterParams{}

			param.TimeStamp = time.Now() // Stop timer
			param.Latency = param.TimeStamp.Sub(start)
			if param.Latency > time.Minute {
				param.Latency = param.Latency.Truncate(time.Second)
			}

			param.ClientIP = c.ClientIP()
			param.Method = c.Request.Method
			param.StatusCode = w.Status()
			param.ErrorMessage = c.Errors.ByType(gin.ErrorTypePrivate).String()
			param.BodySize = w.Size()
			if raw != "" {
				path = path + "?" + raw
			}
			param.Path = path

			respBody := w.body.String()
			// Remove new lines
			respBody = strings.Replace(respBody, "\n", "", -1)
			// Remove double spaces from the string and insert a comma between json objects
			respBody = space.ReplaceAllString(respBody, " ")

			// Log using the params
			var logEvent *zerolog.Event
			if w.Status() >= 500 {
				logEvent = logger.Error()
			} else {
				logEvent = logger.Info()
			}

			logEvent.Str("client_id", param.ClientIP).
				Str("method", param.Method).
				Int("status_code", param.StatusCode).
				Str("path", param.Path).
				Str("latency", param.Latency.String()).
				Int("body_size", param.BodySize).
				Str("body_request", reqBody).
				Str("body_response", respBody).
				Msg(param.ErrorMessage)

			log.Printf("Request:\t%s\n%s", param.Path, reqBody)
			log.Printf("Response:\t%s", respBody)
		}()

	}
}

func GetLogfilePath(c *configs.Config) string {

	logEndpoint := c.LoggingEndpoint

	// Setting the logging path
	if logEndpoint == "" {
		logEndpoint = "./"
	}

	// Logging to a file.
	currentTime := time.Now()
	filename := currentTime.Format("2006-01-02") + ".txt"

	// Check if the directory exists
	if _, err := os.Stat(logEndpoint); os.IsNotExist(err) {
		err = os.MkdirAll(logEndpoint, 0777)
		if err != nil {
			log.Panicf("fail to create log endpoint: %s\n%v\n", logEndpoint, err)
		}
	}

	return logEndpoint + "/" + filename
}

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func readBody(reader io.Reader) string {
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(reader)

	s := buf.String()
	return s
}

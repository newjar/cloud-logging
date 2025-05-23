package cloudlogging

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/logging"
)

type ILogger interface {
	Error(string, map[string]string)
	Warn(string, map[string]string)
	Info(string, map[string]string)
	Debug(string, map[string]string)
	Close() error
}

type Logger struct {
	systemCtx context.Context
	logger    *logging.Logger
	backup    *log.Logger
	gcpClient *logging.Client
}

func NewLogger(ctx context.Context, projectID, loggerName string, backup *log.Logger, labels map[string]string) (ILogger, error) {
	client, err := logging.NewClient(ctx, fmt.Sprintf("projects/%s", projectID))
	if err != nil {
		backup.Printf("WARN: Failed to initialize Google Cloud Logging, falling back to backup logger. Error: %v", err)
		return &Logger{
			systemCtx: ctx,
			logger:    nil,
			backup:    backup,
			gcpClient: nil,
		}, nil
	}

	result := new(Logger)

	logger := client.Logger(loggerName, logging.CommonLabels(labels))

	*result = Logger{
		systemCtx: ctx,
		logger:    logger,
		backup:    backup,
		gcpClient: client,
	}

	return result, nil
}

func payload(msg string, details map[string]string) map[string]string {
	payload := make(map[string]string, len(details)+1)
	payload["msg"] = msg
	for k, v := range details {
		payload[k] = v
	}
	return payload
}

func (l *Logger) log(severity logging.Severity, msg string, details map[string]string) {
	data := payload(msg, details)
	entry := logging.Entry{
		Payload:  data,
		Severity: severity,
	}
	if l.logger == nil || isDone(l.systemCtx) {
		l.backup.Printf("%-10s: %v", severity.String(), data)
	} else {
		l.logger.Log(entry)
	}
}

func (l *Logger) Error(msg string, details map[string]string) {
	l.log(logging.Error, msg, details)
}

func (l *Logger) Warn(msg string, details map[string]string) {
	l.log(logging.Warning, msg, details)
}

func (l *Logger) Info(msg string, details map[string]string) {
	l.log(logging.Info, msg, details)
}

func (l *Logger) Debug(msg string, details map[string]string) {
	l.log(logging.Debug, msg, details)
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}

func (l *Logger) Close() error {
	if l.gcpClient == nil {
		return nil
	}
	return l.gcpClient.Close()
}

package cloudlogging

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/logging"
)

type ILogger interface {
	Error(string, ...string)
	Warn(string, ...string)
	Info(string, ...string)
	Debug(string, ...string)
}

type Logger struct {
	systemCtx context.Context
	logger    *logging.Logger
	backup    *log.Logger
}

func NewLogger(ctx context.Context, projectID, loggerName string, backup *log.Logger, labels ...string) (ILogger, error) {
	client, err := logging.NewClient(ctx, fmt.Sprintf("projects/%s", projectID))
	if err != nil {
		return nil, err
	}

	n := (len(labels) + 1) / 2
	if len(labels)%2 != 0 {
		labels = append(labels, "MISSING")
	}
	commonLabels := make(map[string]string, n)
	for i := 0; i < len(labels); i += 2 {
		commonLabels[labels[i]] = labels[i+1]
	}

	result := new(Logger)

	logger := client.Logger(loggerName, logging.CommonLabels(commonLabels))

	*result = Logger{
		systemCtx: ctx,
		logger:    logger,
		backup:    backup,
	}

	return result, nil
}

func payload(msg string, details ...string) map[string]string {
	n := (len(details) + 1) / 2
	if len(details)%2 != 0 {
		details = append(details, "MISSING")
	}
	payload := make(map[string]string, n+1)
	payload["msg"] = msg
	for i := 0; i < len(details); i += 2 {
		payload[details[i]] = details[i+1]
	}

	return payload
}

func (l *Logger) log(severity logging.Severity, msg string, details ...string) {
	data := payload(msg, details...)
	entry := logging.Entry{
		Payload:  data,
		Severity: severity,
	}
	if isDone(l.systemCtx) {
		l.backup.Printf("%-10s: %v", severity.String(), data)
	} else {
		l.logger.Log(entry)
	}
}

func (l *Logger) Error(msg string, details ...string) {
	l.log(logging.Error, msg, details...)
}

func (l *Logger) Warn(msg string, details ...string) {
	l.log(logging.Warning, msg, details...)
}

func (l *Logger) Info(msg string, details ...string) {
	l.log(logging.Info, msg, details...)
}

func (l *Logger) Debug(msg string, details ...string) {
	l.log(logging.Debug, msg, details...)
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}

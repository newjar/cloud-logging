# cloud-logging
Google Cloud Logging Wrapper

### Requirements
1. Set `GOOGLE_APPLICATION_CREDENTIALS` service account with access to Logging


### Example
```ctx := context.Background()
backupLog := log.New(os.Stdout, "my-backup-log", log.LstdFlags|log.Lmicroseconds)

cloudLogging, err := cloudlogging.NewLogger(ctx, "my-project-id", "my-logging-name", backupLog, map[string]string{"common-key": "common-value"})
if err != nil {
    log.Fatalf("Failed to create logger: %v", err)
}

cloudLogging.Info("i am message from logger", map[string]string{"other_field": "iam value on other_field"})
result on Logging: 
{
  jsonPayload: {
    msg: "i am message from logger",
    other_field: "iam value on other_field"
  },
  labels: {
    "common-key": "common-value"
  }
  severity: "INFO"
}

cloudLogging.Error("someFunction Name", map[string]string{"error": "Error message"})
result on Logging: 
{
  jsonPayload: {
    msg: "someFunction Name",
    error: "Error message"
  },
  labels: {
    "common-key": "common-value"
  }
  severity: "ERROR"
}

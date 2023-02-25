# cloud-logging
Google Cloud Logging Wrapper

### Requirements
1. Set `GOOGLE_APPLICATION_CREDENTIALS` service account with access to Logging


### Example
```ctx := context.Background()
backupLog := log.New(os.Stdout, "my-backup-log", log.LstdFlags|log.Lmicroseconds)

cloudLogging := cloudlogging.NewLogger(ctx, "my-project-id", "my-logging-name")

cloudLogging.Info("i am message from logger", "other_field", "iam value on other_field")
result on Logging: 
{
  jsonPayload: {
    msg: "i am message from logger",
    other_field: "iam value on other_field"
  },
  severity: "INFO"
}

cloudLogging.Error("someFunction Name", "error", err.Error())
result on Logging: 
{
  jsonPayload: {
    msg: "someFunction Name",
    error: "Error message"
  },
  severity: "ERROR"
}

package cloudlogging

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"
)

const (
	validProjectID   = "gcp-project-id" // Replace with a real project ID if you want to test actual GCP interaction
	invalidProjectID = ""               // Or some other invalid project ID
	loggerName       = "test-logger"
)

// TestNewLogger_Success tests the successful creation of a Logger.
// Note: This test might make a real GCP call if validProjectID is a real, configured project.
// For CI/CD or automated environments without credentials, this specific test might need to be skipped
// or use a mocked GCP client, which is beyond the scope of this current task.
func TestNewLogger_Success(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "" && validProjectID == "gcp-project-id" {
		t.Skip("Skipping GCP-dependent test in CI environment without actual project ID")
	}

	ctx := context.Background()
	backup := log.New(os.Stdout, "test-backup ", log.LstdFlags)
	labels := map[string]string{"env": "test"}

	logger, err := NewLogger(ctx, validProjectID, loggerName, backup, labels)

	if err != nil {
		t.Fatalf("NewLogger() error = %v, wantErr %v", err, false)
	}
	if logger == nil {
		t.Fatalf("NewLogger() logger = nil, want not nil")
	}

	// Type assert to access internal fields for testing
	l, ok := logger.(*Logger)
	if !ok {
		t.Fatalf("NewLogger() returned logger is not of type *Logger")
	}

	if l.gcpClient == nil && validProjectID != "" { // if we expected a real client
		t.Errorf("Logger.gcpClient = nil, want not nil for valid project ID")
	}
	if l.backup != backup {
		t.Errorf("Logger.backup not set correctly")
	}
	if l.systemCtx != ctx {
		t.Errorf("Logger.systemCtx not set correctly")
	}
}

// TestNewLogger_Fallback tests the fallback mechanism of NewLogger.
func TestNewLogger_Fallback(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	backup := log.New(&buf, "fallback-test ", log.LstdFlags)
	labels := map[string]string{"env": "fallback"}

	logger, err := NewLogger(ctx, invalidProjectID, loggerName, backup, labels)

	if err != nil {
		t.Fatalf("NewLogger() error = %v, wantErr %v (nil for fallback)", err, false)
	}
	if logger == nil {
		t.Fatalf("NewLogger() logger = nil, want not nil (even on fallback)")
	}

	// Type assert to access internal fields for testing
	l, ok := logger.(*Logger)
	if !ok {
		t.Fatalf("NewLogger() returned logger is not of type *Logger")
	}

	if l.gcpClient != nil {
		t.Errorf("Logger.gcpClient = %v, want nil for fallback", l.gcpClient)
	}
	if l.backup != backup {
		t.Errorf("Logger.backup not set correctly in fallback")
	}

	// Check if the backup logger received a warning message
	output := buf.String()
	expectedWarning := "WARN: Failed to initialize Google Cloud Logging"
	if !strings.Contains(output, expectedWarning) {
		t.Errorf("Backup logger output does not contain expected warning.\nGot: %s\nWant: %s", output, expectedWarning)
	}
}

// TestLoggingMethods tests the logging methods (Info, Warn, Error, Debug).
func TestLoggingMethods(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	backup := log.New(&buf, "", 0) // No prefix for simpler matching
	labels := map[string]string{}

	// Use invalid project ID to ensure backup logger is used
	logger, _ := NewLogger(ctx, invalidProjectID, loggerName, backup, labels)

	// Cast to *Logger to access gcpClient and set it to nil explicitly for this test's purpose
	// This ensures we are testing the backup path regardless of NewLogger behavior with "" projectID
	if l, ok := logger.(*Logger); ok {
		l.logger = nil // Force use of backup logger
	} else {
		t.Fatal("Could not cast logger to *Logger")
	}


	testCases := []struct {
		level    string
		logFunc  func(msg string, details map[string]string)
		severity string // Expected severity string in backup log
	}{
		{"Info", logger.Info, "INFO"},
		{"Warn", logger.Warn, "WARNING"},
		{"Error", logger.Error, "ERROR"},
		{"Debug", logger.Debug, "DEBUG"},
	}

	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			buf.Reset() // Clear buffer for each test case
			msg := "This is a " + tc.level + " message"
			details := map[string]string{"key1": "value1", "source": "test"}
			
			logger.(*Logger).log(logSeverityFromString(tc.severity), msg, details) // calling internal log directly for consistent output format with backup

			output := buf.String()

			if !strings.Contains(output, tc.severity) {
				t.Errorf("Expected log output to contain severity '%s', got: %s", tc.severity, output)
			}
			if !strings.Contains(output, msg) {
				t.Errorf("Expected log output to contain message '%s', got: %s", msg, output)
			}
			if !strings.Contains(output, "key1:value1") && !strings.Contains(output, "key1=value1") { // map format can vary
				t.Errorf("Expected log output to contain details 'key1:value1' or 'key1=value1', got: %s", output)
			}
			if !strings.Contains(output, "source:test") && !strings.Contains(output, "source=test") {
				t.Errorf("Expected log output to contain details 'source:test' or 'source=test', got: %s", output)
			}
		})
	}
}

// Helper to convert string severity to logging.Severity for TestLoggingMethods
func logSeverityFromString(level string) log.Lvl {
	// This is a simplified mapping for the backup logger's Printf format.
	// The actual cloud.google.com/go/logging.Severity is not used directly by backup logger.
	// We are checking the string representation.
	// For the purpose of this test, we only need to ensure the string appears.
	// The backup logger in the code uses severity.String(), so we match that.
	// This helper is actually not needed if we check for the string directly.
	// The backup format is: l.backup.Printf("%-10s: %v", severity.String(), data)
	// So we just need to ensure "INFO      :", "WARNING   :", etc.
	// For simplicity, direct string check in test is fine.
	// This function is not used due to direct string check.
	return 0 // Placeholder, not actually used
}


// TestClose_ClientExists tests the Close method when gcpClient is not nil.
// This test relies on TestNewLogger_Success to create a logger with a potentially real client.
func TestClose_ClientExists(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "" && validProjectID == "gcp-project-id" {
		t.Skip("Skipping GCP-dependent test in CI environment without actual project ID for Close with client")
	}

	ctx := context.Background()
	backup := log.New(os.Stdout, "test-close-client ", log.LstdFlags)
	
	// Assuming NewLogger successfully creates a client with validProjectID
	logger, err := NewLogger(ctx, validProjectID, loggerName, backup, nil)
	if err != nil {
		// If NewLogger itself fails (e.g. no credentials for validProjectID), we can't test this case.
		t.Fatalf("NewLogger() failed with valid project ID, cannot proceed to test Close(): %v", err)
	}
	if logger == nil {
		t.Fatal("NewLogger returned nil logger with valid project ID")
	}
	
	l, ok := logger.(*Logger)
	if !ok {
		t.Fatal("Logger is not of type *Logger")
	}

	// Only proceed if gcpClient was actually initialized
	if l.gcpClient == nil && validProjectID != "" {
		t.Logf("gcpClient is nil even with a supposedly valid project ID. This might be due to environment/auth issues. Skipping the core part of TestClose_ClientExists.")
		// We can still call Close and it should be a no-op or handled gracefully.
		closeErr := logger.Close()
		if closeErr != nil {
			t.Errorf("Close() error = %v, want nil (when gcpClient was already nil)", closeErr)
		}
		return
	}
    if l.gcpClient == nil && validProjectID == "" {
        // This case should not happen based on NewLogger logic, but if it did, it's a nil client case
        closeErr := logger.Close()
		if closeErr != nil {
			t.Errorf("Close() error = %v, want nil (when gcpClient was nil due to empty project ID)", closeErr)
		}
        return
    }


	closeErr := logger.Close()
	if closeErr != nil {
		t.Errorf("Close() error = %v, want nil", closeErr)
	}
	// Optionally, one might try to use the logger again to see if it errors out,
	// but the definition of "Close" for the client might not prevent further Log calls
	// from being attempted (they might just fail).
}

// TestClose_NilClient tests the Close method when gcpClient is nil.
func TestClose_NilClient(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	backup := log.New(&buf, "nil-client-close-test ", log.LstdFlags)

	// Create a logger that will have a nil gcpClient by providing an invalid project ID
	logger, _ := NewLogger(ctx, invalidProjectID, loggerName, backup, nil)
	if logger == nil {
		t.Fatal("NewLogger returned nil for nil-client test")
	}
	
	l, ok := logger.(*Logger)
	if !ok {
		t.Fatal("Logger is not of type *Logger")
	}
	if l.gcpClient != nil {
		t.Fatalf("gcpClient is not nil for a logger expected to have a nil client. Value: %v", l.gcpClient)
	}

	err := logger.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil for nil gcpClient", err)
	}
}

// TestPayloadFunction ensures the payload function correctly creates maps.
func TestPayloadFunction(t *testing.T) {
	msg := "test message"
	details := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	p := payload(msg, details)

	if p["msg"] != msg {
		t.Errorf("payload msg = %s, want %s", p["msg"], msg)
	}
	if p["key1"] != "value1" {
		t.Errorf("payload key1 = %s, want value1", p["key1"])
	}
	if p["key2"] != "value2" {
		t.Errorf("payload key2 = %s, want value2", p["key2"])
	}
	if len(p) != 3 {
		t.Errorf("payload len = %d, want 3", len(p))
	}

	detailsNil := map[string]string{}
	pNil := payload(msg, detailsNil)
	if pNil["msg"] != msg {
		t.Errorf("payload msg (nil details) = %s, want %s", pNil["msg"], msg)
	}
	if len(pNil) != 1 {
		t.Errorf("payload len (nil details) = %d, want 1", len(pNil))
	}
}

// TestIsDone (Optional, simple test for a simple function)
func TestIsDone(t *testing.T) {
	ctxDone, cancel := context.WithCancel(context.Background())
	cancel() // make it done

	if !isDone(ctxDone) {
		t.Error("isDone(ctxDone) = false, want true")
	}

	ctxNotDone := context.Background()
	if isDone(ctxNotDone) {
		t.Error("isDone(ctxNotDone) = true, want false")
	}
}

// TestLoggingMethods_ActualCalls tests the actual logging methods (Info, Warn, Error, Debug)
// by calling them directly, not the internal log() method.
func TestLoggingMethods_ActualCalls(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	// Using a prefix to ensure it's part of the output, and flags for timestamp.
	backup := log.New(&buf, "BKP: ", log.LstdFlags)
	labels := map[string]string{}

	// Use invalid project ID to ensure backup logger is used
	logger, _ := NewLogger(ctx, invalidProjectID, loggerName, backup, labels)

	if l, ok := logger.(*Logger); ok {
		l.logger = nil // Force use of backup logger for predictability
	} else {
		t.Fatal("Could not cast logger to *Logger")
	}

	testCases := []struct {
		level    string
		logFunc  func(msg string, details map[string]string)
		severity string // Expected severity string in backup log
	}{
		{"Info", logger.Info, "INFO"},
		{"Warn", logger.Warn, "WARNING"},
		{"Error", logger.Error, "ERROR"},
		{"Debug", logger.Debug, "DEBUG"},
	}

	for _, tc := range testCases {
		t.Run(tc.level+"_ActualCall", func(t *testing.T) {
			buf.Reset()
			msg := "Actual " + tc.level + " call"
			details := map[string]string{"detailKey": "detailValue"}

			tc.logFunc(msg, details) // Call the actual Info, Warn, Error, Debug

			output := buf.String()

			if !strings.Contains(output, "BKP: ") {
				t.Errorf("Expected log output to contain backup prefix 'BKP: ', got: %s", output)
			}
			if !strings.Contains(output, tc.severity) {
				t.Errorf("Expected log output to contain severity '%s', got: %s", tc.severity, output)
			}
			if !strings.Contains(output, msg) {
				t.Errorf("Expected log output to contain message '%s', got: %s", msg, output)
			}
			// Map formatting in log output can be tricky (e.g. "map[key:value]" or "key:value").
			// Be flexible with the check or serialize consistently.
			// The backup logger prints map as `map[detailKey:detailValue msg:Actual Info call]`
			// So we check for parts of it.
			if !strings.Contains(output, "detailKey:detailValue") {
				t.Errorf("Expected log output to contain details 'detailKey:detailValue', got: %s", output)
			}
		})
	}
}

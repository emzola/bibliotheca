package jsonlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestJSONLogger(t *testing.T) {
	var logBuffer bytes.Buffer

	t.Run("ANYLevel", func(t *testing.T) {
		message := "Any log"
		// Use LevelFatal as argument in New() function so that when running tests
		// the log is written to the standard output but not printed out
		l := New(os.Stdout, LevelFatal)
		expected := 1
		_, err := l.print(LevelInfo, message, nil)
		if err != nil {
			t.Fatal(err)
		}
		l.out = &logBuffer
		lines := bytes.Split(logBuffer.Bytes(), []byte("\n"))
		if len(lines) != expected {
			t.Errorf("Expected %d log lines, got %d instead\n", expected, len(lines))
		}
		logBuffer.Reset()
	})

	t.Run("INFOLevel", func(t *testing.T) {
		message := "INFO log"
		properties := map[string]string{
			"addr": "8080",
			"env":  "development",
		}
		// Use LevelFatal as argument in New() function so that when running tests
		// the log is written to the standard output but not printed out
		l := New(os.Stdout, LevelFatal)
		var output string
		l.PrintInfo(message, properties)
		l.out = &logBuffer
		json.NewDecoder(&logBuffer).Decode(&output)
		if output != logBuffer.String() {
			t.Errorf("Expected %s, got %s instead\n", output, &logBuffer)
		}
		logBuffer.Reset()
	})

	t.Run("ERRORLevel", func(t *testing.T) {
		message := "ERROR log"
		err := fmt.Errorf("%s", message)
		// Use LevelFatal as argument in New() function so that when running tests
		// the log is written to the standard output but not printed out
		l := New(os.Stdout, LevelFatal)
		var output string
		l.PrintError(err, nil)
		l.out = &logBuffer
		json.NewDecoder(&logBuffer).Decode(&output)
		if output != logBuffer.String() {
			t.Errorf("Expected %s, got %s instead\n", output, &logBuffer)
		}
		logBuffer.Reset()
	})

	t.Run("FATALLevel", func(t *testing.T) {
		message := "FATAL log"
		err := fmt.Errorf("%s", message)
		// Use LevelFatal as argument in New() function so that when running tests
		// the log is written to the standard output but not printed out
		l := New(os.Stdout, LevelFatal)
		var output string
		l.PrintError(err, nil)
		l.out = &logBuffer
		json.NewDecoder(&logBuffer).Decode(&output)
		if output != logBuffer.String() {
			t.Errorf("Expected %s, got %s instead\n", output, &logBuffer)
		}
		logBuffer.Reset()
	})
}

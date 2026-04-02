package logger

import "testing"

func TestNewLoggerAndFields(t *testing.T) {
	log := New("debug", "json")
	if log == nil {
		t.Fatal("logger should not be nil")
	}
	entry := log.WithField("k", "v")
	if entry == nil {
		t.Fatal("entry should not be nil")
	}
	entry2 := log.WithFields(Fields{"k": "v"})
	if entry2 == nil {
		t.Fatal("entry2 should not be nil")
	}
}

package model

import "testing"

func TestIsValidTaskStatus(t *testing.T) {
	task := &Task{}
	if !task.IsValidTaskStatus("CREATED") {
		t.Fatal("CREATED should be valid")
	}
	if !task.IsValidTaskStatus("WORKING_ON") {
		t.Fatal("WORKING_ON should be valid")
	}
	if !task.IsValidTaskStatus("CLOSED") {
		t.Fatal("CLOSED should be valid")
	}
	if task.IsValidTaskStatus("UNKNOWN") {
		t.Fatal("UNKNOWN should be invalid")
	}
}

package main

import "testing"

func TestUsageError(t *testing.T) {
err := usageError("test")
if err.Error() != "usage error: test" {
t.Errorf("Expected usage error: test, got %s", err.Error())
}
}

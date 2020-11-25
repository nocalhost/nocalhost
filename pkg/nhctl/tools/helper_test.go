package tools

import (
	"context"
	"testing"
)

func TestExecCommand(t *testing.T) {
	ExecCommand(context.TODO(), true, "git", "clone", "adfa")
}

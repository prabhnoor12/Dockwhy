package output

import (
	"strings"
	"testing"

	"github.com/dockwhy/dockwhy/internal/diagnosis"
	"github.com/dockwhy/dockwhy/internal/docker"
)

func TestJSONUsesStableLowercaseSchema(t *testing.T) {
	var out strings.Builder
	result := diagnosis.Result{Container: docker.Container{ID: "abc", MemoryLimit: 1024}, Reason: "clean exit"}
	if err := JSON(&out, result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"id": "abc"`) || strings.Contains(out.String(), `"ID"`) {
		t.Fatalf("unexpected JSON schema: %s", out.String())
	}
}

func TestTextMarksTruncatedLogs(t *testing.T) {
	var out strings.Builder
	result := diagnosis.Result{Container: docker.Container{Name: "api"}, Logs: "last line", LogsTruncated: true}
	if err := Text(&out, result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[output truncated]") {
		t.Fatalf("missing truncation marker: %s", out.String())
	}
}

package export

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nad/pkgview/internal/model"
)

func TestTXT(t *testing.T) {
	out := string(TXT([]model.Package{{Name: "gh", Version: "2.0.0", Source: model.SourceHomebrew}}))
	if want := "gh\t2.0.0\thomebrew\n"; out != want {
		t.Fatalf("TXT() = %q, want %q", out, want)
	}
}

func TestTXTEmpty(t *testing.T) {
	out := string(TXT(nil))
	if out != "" {
		t.Fatalf("TXT(nil) = %q, want empty string", out)
	}
}

func TestJSON(t *testing.T) {
	out, err := JSON([]model.Package{{Name: "gh", Version: "2.0.0", Source: model.SourceHomebrew}})
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	if !strings.Contains(string(out), `"name": "gh"`) {
		t.Fatalf("JSON() output = %s", out)
	}
}

func TestWriteJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "packages.json")
	if err := WriteJSON(path, []model.Package{{Name: "gh", Source: model.SourceHomebrew}}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Fatalf("file missing trailing newline: %q", data)
	}
}

func TestWriteTXT(t *testing.T) {
	path := filepath.Join(t.TempDir(), "packages.txt")
	if err := WriteTXT(path, []model.Package{{Name: "gh", Version: "2.0.0", Source: model.SourceHomebrew}}); err != nil {
		t.Fatalf("WriteTXT() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if want := "gh\t2.0.0\thomebrew\n"; string(data) != want {
		t.Fatalf("WriteTXT() file = %q, want %q", data, want)
	}
}

func TestWriteJSONMarshalError(t *testing.T) {
	original := marshalJSON
	t.Cleanup(func() {
		marshalJSON = original
	})

	marshalJSON = func([]model.Package) ([]byte, error) {
		return nil, errors.New("boom")
	}

	err := WriteJSON(filepath.Join(t.TempDir(), "packages.json"), []model.Package{{Name: "gh"}})
	if err == nil || !strings.Contains(err.Error(), "marshal export json: boom") {
		t.Fatalf("WriteJSON() error = %v, want wrapped marshal error", err)
	}
}

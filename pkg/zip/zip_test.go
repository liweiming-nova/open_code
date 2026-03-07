package zip

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestUnzip_ExtractsFilesAndCallback(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	srcZip := filepath.Join(tmpDir, "sample.zip")
	dstDir := filepath.Join(tmpDir, "out")

	createZipFile(t, srcZip, map[string]string{
		"dir/":      "",
		"dir/a.txt": "hello",
		"b.txt":     "world",
	})

	type progress struct {
		processed int
		total     int
		name      string
		isDir     bool
	}
	var calls []progress
	err := Unzip(srcZip, dstDir, func(processed, total int, fileName string, isDir bool) {
		calls = append(calls, progress{processed: processed, total: total, name: fileName, isDir: isDir})
	})
	if err != nil {
		t.Fatalf("Unzip() error = %v", err)
	}

	gotA, err := os.ReadFile(filepath.Join(dstDir, "dir", "a.txt"))
	if err != nil {
		t.Fatalf("read extracted dir/a.txt failed: %v", err)
	}
	if string(gotA) != "hello" {
		t.Fatalf("dir/a.txt content = %q, want %q", string(gotA), "hello")
	}

	gotB, err := os.ReadFile(filepath.Join(dstDir, "b.txt"))
	if err != nil {
		t.Fatalf("read extracted b.txt failed: %v", err)
	}
	if string(gotB) != "world" {
		t.Fatalf("b.txt content = %q, want %q", string(gotB), "world")
	}

	if len(calls) != 3 {
		t.Fatalf("callback count = %d, want 3", len(calls))
	}
	last := calls[len(calls)-1]
	if last.processed != 3 || last.total != 3 {
		t.Fatalf("last callback = (%d/%d), want (3/3)", last.processed, last.total)
	}
}

func TestUnzip_RejectsZipSlip(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	srcZip := filepath.Join(tmpDir, "zipslip.zip")
	dstDir := filepath.Join(tmpDir, "out")

	createZipFile(t, srcZip, map[string]string{
		"../evil.txt": "bad",
	})

	if err := Unzip(srcZip, dstDir, nil); err == nil {
		t.Fatal("Unzip() expected error for zip slip payload, got nil")
	}
}

func createZipFile(t *testing.T, path string, entries map[string]string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip file failed: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for name, body := range entries {
		h := &zip.FileHeader{Name: name}
		if name[len(name)-1] == '/' {
			h.SetMode(os.ModeDir | 0o755)
		} else {
			h.SetMode(0o644)
		}

		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatalf("create zip header for %q failed: %v", name, err)
		}
		if body != "" {
			if _, err := w.Write([]byte(body)); err != nil {
				t.Fatalf("write zip entry %q failed: %v", name, err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer failed: %v", err)
	}
}

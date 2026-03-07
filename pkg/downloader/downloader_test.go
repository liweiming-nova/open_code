package downloader

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDownloader_Start_DownloadsFullFile(t *testing.T) {
	t.Parallel()

	data := []byte(strings.Repeat("abc123", 1024))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer server.Close()

	dst := filepath.Join(t.TempDir(), "full.bin")
	d := New(server.URL, dst, server.Client())
	d.SetProgressInterval(1 * time.Millisecond)

	var mu sync.Mutex
	progressCalls := 0
	var lastDownloaded int64
	var lastTotal int64
	d.OnProgress(func(downloaded, total int64, rate string) {
		mu.Lock()
		defer mu.Unlock()
		progressCalls++
		lastDownloaded = downloaded
		lastTotal = total
		if rate == "" {
			t.Errorf("rate should not be empty")
		}
	})

	if err := d.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("downloaded content mismatch")
	}

	mu.Lock()
	defer mu.Unlock()
	if progressCalls == 0 {
		t.Fatal("OnProgress callback was not called")
	}
	if lastDownloaded != int64(len(data)) {
		t.Fatalf("last downloaded = %d, want %d", lastDownloaded, len(data))
	}
	if lastTotal != int64(len(data)) {
		t.Fatalf("last total = %d, want %d", lastTotal, len(data))
	}
}

func TestDownloader_Start_ResumesWithRange(t *testing.T) {
	t.Parallel()

	data := []byte(strings.Repeat("resume-data-", 2048))

	var mu sync.Mutex
	sawRange := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}

		mu.Lock()
		sawRange = true
		mu.Unlock()

		const prefix = "bytes="
		if !strings.HasPrefix(rangeHeader, prefix) || !strings.HasSuffix(rangeHeader, "-") {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		startStr := strings.TrimSuffix(strings.TrimPrefix(rangeHeader, prefix), "-")
		start, err := strconv.Atoi(startStr)
		if err != nil || start < 0 || start > len(data) {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		if start == len(data) {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		part := data[start:]
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(data)-1, len(data)))
		w.Header().Set("Content-Length", strconv.Itoa(len(part)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(part)
	}))
	defer server.Close()

	dst := filepath.Join(t.TempDir(), "resume.bin")
	offset := len(data) / 3
	if err := os.WriteFile(dst, data[:offset], 0o644); err != nil {
		t.Fatalf("prepare partial file failed: %v", err)
	}

	d := New(server.URL, dst, server.Client())
	if err := d.Start(); err != nil {
		t.Fatalf("Start() resume error = %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("resumed content mismatch")
	}

	mu.Lock()
	defer mu.Unlock()
	if !sawRange {
		t.Fatal("expected request with Range header, but none observed")
	}
}

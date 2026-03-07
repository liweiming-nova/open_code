package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	appErr "github.com/liweiming-nova/open_code/errors"
)

const (
	defaultBufferSize       = 32 * 1024
	defaultProgressInterval = 500 * time.Millisecond
)

// ProgressFunc receives bytes downloaded, total bytes and current rate.
type ProgressFunc func(downloaded, total int64, rate string)

// Downloader provides single-connection download with resume support.
type Downloader struct {
	url              string
	destination      string
	client           *http.Client
	onProgress       ProgressFunc
	progressInterval time.Duration
}

// New creates a downloader.
func New(url, destination string, client *http.Client) *Downloader {
	if client == nil {
		client = http.DefaultClient
	}
	return &Downloader{
		url:              strings.TrimSpace(url),
		destination:      strings.TrimSpace(destination),
		client:           client,
		progressInterval: defaultProgressInterval,
	}
}

// OnProgress sets the progress callback.
func (d *Downloader) OnProgress(cb ProgressFunc) {
	d.onProgress = cb
}

// SetProgressInterval sets callback interval. Non-positive value is ignored.
func (d *Downloader) SetProgressInterval(interval time.Duration) {
	if interval > 0 {
		d.progressInterval = interval
	}
}

// Start downloads the file. Existing file is resumed when server supports ranges.
func (d *Downloader) Start() error {
	return d.StartContext(context.Background())
}

// StartContext downloads with cancellation support.
func (d *Downloader) StartContext(ctx context.Context) error {
	if d.url == "" {
		return appErr.ErrEmptyURL
	}
	if d.destination == "" {
		return fmt.Errorf("empty destination")
	}
	if err := os.MkdirAll(filepath.Dir(d.destination), 0o755); err != nil {
		return err
	}

	localSize := int64(0)
	if fi, err := os.Stat(d.destination); err == nil {
		localSize = fi.Size()
	} else if !os.IsNotExist(err) {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.url, nil)
	if err != nil {
		return err
	}
	if localSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", localSize))
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		if d.onProgress != nil {
			d.onProgress(localSize, localSize, "0 B/s")
		}
		return nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	appendMode := localSize > 0 && resp.StatusCode == http.StatusPartialContent
	if !appendMode {
		localSize = 0
	}

	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(d.destination, flags, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	total := resp.ContentLength
	if total >= 0 {
		total += localSize
	}
	downloaded := localSize

	buffer := make([]byte, defaultBufferSize)
	lastTick := time.Now()
	lastTickBytes := downloaded
	if d.onProgress != nil {
		d.onProgress(downloaded, total, "0 B/s")
	}

	for {
		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			if _, err = file.Write(buffer[:n]); err != nil {
				return err
			}
			downloaded += int64(n)
		}

		now := time.Now()
		if d.onProgress != nil && (now.Sub(lastTick) >= d.progressInterval || readErr == io.EOF) {
			deltaBytes := downloaded - lastTickBytes
			deltaSeconds := now.Sub(lastTick).Seconds()
			rate := "0 B/s"
			if deltaSeconds > 0 {
				rate = formatRate(float64(deltaBytes) / deltaSeconds)
			}
			d.onProgress(downloaded, total, rate)
			lastTick = now
			lastTickBytes = downloaded
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	return nil
}

func formatRate(bytesPerSecond float64) string {
	const (
		KB = 1024.0
		MB = KB * 1024.0
		GB = MB * 1024.0
	)

	switch {
	case bytesPerSecond >= GB:
		return fmt.Sprintf("%.2f GB/s", bytesPerSecond/GB)
	case bytesPerSecond >= MB:
		return fmt.Sprintf("%.2f MB/s", bytesPerSecond/MB)
	case bytesPerSecond >= KB:
		return fmt.Sprintf("%.2f KB/s", bytesPerSecond/KB)
	default:
		return fmt.Sprintf("%.2f B/s", bytesPerSecond)
	}
}

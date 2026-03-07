package zip

import (
	"fmt"
	"testing"
)

func TestUnZip(t *testing.T) {
	err := Unzip("data.zip", "./output", func(processed, total int, fileName string, isDir bool) {
		fmt.Printf("[%d/%d] %s (%s)\n", processed, total, fileName,
			map[bool]string{true: "目录", false: "文件"}[isDir])
	})
	if err != nil {
		t.Fatalf("unzip error: %v", err)
	}
}

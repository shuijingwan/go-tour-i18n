package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSurfaceTransportCLIContract(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		{}, {"--locale", "zh-CN", "--batch-id", "chatgpt-zh-CN-001"},
		{"--locale", "zh-CN", "--batch-id", "chatgpt-zh-CN-001", "--discover", "--record"},
		{"--locale", "zh-CN", "--batch-id", "chatgpt-zh-CN-001", "--discover", "--input", "request.json"},
		{"--locale", "zh-CN", "--batch-id", "chatgpt-zh-CN-001", "--input", "request.json", "extra"},
	} {
		if err := surfaceTransportCommand(root, nil, args); err == nil || !strings.Contains(err.Error(), "usage:") {
			t.Fatalf("invalid discovery/preflight/record contract: %v: %v", args, err)
		}
	}
	request := filepath.Join(root, "request.json")
	if err := os.WriteFile(request, []byte(`{"schema":"go-learning/surface-revision-transport/v1","unknown":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--locale", "zh-CN", "--batch-id", "chatgpt-zh-CN-001", "--input", request}
	if err := surfaceTransportCommand(root, nil, args); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatal("noncanonical request accepted", err)
	}
	link := filepath.Join(root, "link.json")
	if err := os.Symlink(request, link); err != nil {
		t.Fatal(err)
	}
	args[len(args)-1] = link
	if err := surfaceTransportCommand(root, nil, args); err == nil {
		t.Fatal("symlink request accepted")
	}
	if _, err := os.Stat(filepath.Join(root, "data")); !os.IsNotExist(err) {
		t.Fatal("invalid CLI mutated formal authority", err)
	}
}

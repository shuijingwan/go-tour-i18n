package i18n

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"strings"
	"testing"
	"time"
)

func TestDeterministicTransportBundleAndExactInventory(t *testing.T) {
	entries := []TransportBundleEntry{{Path: "z.txt", Data: []byte("z")}, {Path: "a.txt", Data: []byte("a")}}
	first, err := WriteDeterministicTransportBundle([]byte("{}\n"), entries)
	if err != nil {
		t.Fatal(err)
	}
	second, err := WriteDeterministicTransportBundle([]byte("{}\n"), entries)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("deterministic bundle mismatch: err=%v", err)
	}
	files, err := ReadTransportBundle(first, 4, 1024)
	if err != nil {
		t.Fatal(err)
	}
	inventory := []TransportBundleFile{NewTransportBundleFile("a.txt", "", []byte("a")), NewTransportBundleFile("z.txt", "", []byte("z"))}
	if err := ValidateTransportBundleInventory(files, inventory, true); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTransportBundleInventory(files, inventory[:1], true); err == nil || !strings.Contains(err.Error(), "file set mismatch") {
		t.Fatalf("unexpected entry was accepted: %v", err)
	}
}

func TestReadTransportBundleRejectsTraversalAndSymlink(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		mode fs.FileMode
	}{
		{name: "traversal", path: "../escape", mode: 0644},
		{name: "absolute", path: "/escape", mode: 0644},
		{name: "symlink", path: "link", mode: fs.ModeSymlink | 0777},
	} {
		t.Run(test.name, func(t *testing.T) {
			var buffer bytes.Buffer
			writer := zip.NewWriter(&buffer)
			manifest := &zip.FileHeader{Name: "manifest.json", Method: zip.Store}
			manifest.SetMode(0644)
			manifest.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
			file, _ := writer.CreateHeader(manifest)
			_, _ = file.Write([]byte("{}\n"))
			header := &zip.FileHeader{Name: test.path, Method: zip.Store}
			header.SetMode(test.mode)
			header.Modified = manifest.Modified
			file, _ = writer.CreateHeader(header)
			_, _ = file.Write([]byte("payload"))
			_ = writer.Close()
			if _, err := ReadTransportBundle(buffer.Bytes(), 4, 1024); err == nil {
				t.Fatal("unsafe ZIP entry was accepted")
			}
		})
	}
}

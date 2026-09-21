package i18n

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"
)

// TransportBundleEntry is one regular file in a deterministic transport ZIP.
// Transport bundles are never semantic authority or review evidence.
type TransportBundleEntry struct {
	Path string
	Data []byte
}

// TransportBundleFile is the shared, machine-readable file inventory shape.
type TransportBundleFile struct {
	BundlePath     string `json:"bundle_path"`
	RepositoryPath string `json:"repository_path,omitempty"`
	SHA256         string `json:"sha256"`
	Size           int64  `json:"size,omitempty"`
}

func NewTransportBundleFile(bundlePath, repositoryPath string, data []byte) TransportBundleFile {
	return TransportBundleFile{
		BundlePath: bundlePath, RepositoryPath: repositoryPath,
		SHA256: sum(data), Size: int64(len(data)),
	}
}

func validateTransportBundlePath(name string) error {
	if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || path.Clean(name) != name || name == "." || strings.HasPrefix(name, "../") || strings.Contains(name, "/../") {
		return fmt.Errorf("unsafe bundle path %q", name)
	}
	return nil
}

// WriteDeterministicTransportBundle writes byte-stable, uncompressed ZIP
// bytes. manifest.json is always first; all other entries are path-sorted.
func WriteDeterministicTransportBundle(manifest []byte, entries []TransportBundleEntry) ([]byte, error) {
	all := make([]TransportBundleEntry, 0, len(entries)+1)
	all = append(all, TransportBundleEntry{Path: "manifest.json", Data: manifest})
	all = append(all, entries...)
	seen := map[string]bool{}
	for _, entry := range all {
		if err := validateTransportBundlePath(entry.Path); err != nil {
			return nil, err
		}
		if seen[entry.Path] {
			return nil, fmt.Errorf("duplicate bundle path %q", entry.Path)
		}
		seen[entry.Path] = true
	}
	sort.Slice(all[1:], func(i, j int) bool { return all[i+1].Path < all[j+1].Path })

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range all {
		header := &zip.FileHeader{Name: entry.Path, Method: zip.Store}
		header.SetMode(0644)
		header.Modified = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
		file, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := file.Write(entry.Data); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// ReadTransportBundle rejects directories, symlinks, path traversal,
// duplicate/unexpected ZIP metadata, and oversized members before returning
// the exact regular-file map.
func ReadTransportBundle(data []byte, maxFiles int, maxTotalBytes int64) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open transport bundle: %w", err)
	}
	if len(reader.File) == 0 || len(reader.File) > maxFiles {
		return nil, fmt.Errorf("transport bundle file count %d is outside 1..%d", len(reader.File), maxFiles)
	}
	files := make(map[string][]byte, len(reader.File))
	var total int64
	for _, file := range reader.File {
		if err := validateTransportBundlePath(file.Name); err != nil {
			return nil, err
		}
		if file.FileInfo().IsDir() || !file.Mode().IsRegular() {
			return nil, fmt.Errorf("bundle entry %q is not a regular file", file.Name)
		}
		if file.Method != zip.Store {
			return nil, fmt.Errorf("bundle entry %q uses unsupported compression method", file.Name)
		}
		if _, exists := files[file.Name]; exists {
			return nil, fmt.Errorf("duplicate bundle entry %q", file.Name)
		}
		if int64(file.UncompressedSize64) > maxTotalBytes-total {
			return nil, fmt.Errorf("transport bundle exceeds %d bytes", maxTotalBytes)
		}
		stream, err := file.Open()
		if err != nil {
			return nil, err
		}
		payload, readErr := io.ReadAll(io.LimitReader(stream, maxTotalBytes-total+1))
		closeErr := stream.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		total += int64(len(payload))
		if total > maxTotalBytes {
			return nil, fmt.Errorf("transport bundle exceeds %d bytes", maxTotalBytes)
		}
		files[file.Name] = payload
	}
	if _, ok := files["manifest.json"]; !ok {
		return nil, fmt.Errorf("transport bundle is missing manifest.json")
	}
	return files, nil
}

func ValidateTransportBundleInventory(files map[string][]byte, inventory []TransportBundleFile, allowManifest bool) error {
	want := make(map[string]TransportBundleFile, len(inventory)+1)
	if allowManifest {
		want["manifest.json"] = TransportBundleFile{BundlePath: "manifest.json"}
	}
	for _, item := range inventory {
		if err := validateTransportBundlePath(item.BundlePath); err != nil {
			return err
		}
		if item.BundlePath == "manifest.json" || want[item.BundlePath].BundlePath != "" {
			return fmt.Errorf("duplicate inventory path %q", item.BundlePath)
		}
		if !validSHA256(item.SHA256) || item.Size < 0 {
			return fmt.Errorf("invalid inventory identity for %q", item.BundlePath)
		}
		want[item.BundlePath] = item
	}
	if len(files) != len(want) {
		return fmt.Errorf("bundle file set mismatch: got %d files, want %d", len(files), len(want))
	}
	for name, data := range files {
		item, ok := want[name]
		if !ok {
			return fmt.Errorf("unexpected bundle entry %q", name)
		}
		if name == "manifest.json" {
			continue
		}
		if int64(len(data)) != item.Size || sum(data) != item.SHA256 {
			return fmt.Errorf("bundle entry identity mismatch: %s", name)
		}
	}
	return nil
}

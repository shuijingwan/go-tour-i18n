package i18n

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"
)

var historicalBytesImplementationPaths = historicalBytesVerifierImplementationPaths()

func historicalBytesVerifierImplementationPaths() []string {
	paths := []string{
		"internal/i18n/historical_construction_bytes.go", "internal/i18n/historical_construction_graph.go",
		"internal/i18n/historical_construction_io.go", "internal/i18n/historical_construction_authority.go",
		"internal/i18n/historical_construction_transaction.go", "internal/i18n/historical_construction_resolver.go",
	}
	// These sources own the versioned graph/declaration readers and the bounded
	// archive/import checks called above. This list belongs to the NEW evidence
	// verifier, never to the fixed old approved producer inventory.
	paths = append(paths, transportRecoveryImplementationPaths...)
	for path := range approvedConstructionDeclarationsV1 {
		paths = append(paths, path)
	}
	paths = append(paths, "internal/i18n/generation_bundle.go", "internal/i18n/generation_import_archive.go", "internal/i18n/transport_bundle.go", "internal/i18n/translation_package.go", "internal/contentidentity/strict_json.go", "cmd/tour-i18n/historical_construction_bytes.go", "cmd/tour-i18n/main.go")
	seen := map[string]bool{}
	unique := []string{}
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}
	sort.Strings(unique)
	return unique
}

func historicalBytesAbsolutePath(p string) bool {
	if !utf8.ValidString(p) || !filepath.IsAbs(p) || filepath.Clean(p) != p {
		return false
	}
	for _, r := range p {
		if unicode.IsControl(r) {
			return false
		}
	}
	for _, part := range strings.Split(p, string(filepath.Separator)) {
		if part == ".." {
			return false
		}
	}
	return true
}

func historicalBytesNoSymlinks(p string) error {
	if !historicalBytesAbsolutePath(p) {
		return hbError("UNSAFE_PATH", "canonical absolute local path required")
	}
	for path := p; ; path = filepath.Dir(path) {
		info, err := os.Lstat(path)
		if err != nil {
			return hbWrap("UNSAFE_PATH", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return hbError("UNSAFE_PATH", "symlink component forbidden")
		}
		if path != p && !info.IsDir() {
			return hbError("UNSAFE_PATH", "non-directory parent")
		}
		if path == filepath.Dir(path) {
			break
		}
	}
	return nil
}

func historicalBytesStat(info os.FileInfo, b []byte) (HistoricalBytesObservation, error) {
	s, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || s.Nlink != 1 {
		return HistoricalBytesObservation{}, hbError("INVALID_ACQUISITION", "regular single-link file required")
	}
	return HistoricalBytesObservation{sum(b), int64(len(b)), uint64(s.Dev), s.Ino, s.Mode, uint64(s.Nlink), s.Mtim.Sec*1e9 + s.Mtim.Nsec, s.Ctim.Sec*1e9 + s.Ctim.Nsec}, nil
}

// Bound and observe the exact descriptor, including nlink/type both before and
// after the read. Hash equality is not a license to follow a source symlink.
func readHistoricalBytesExternal(root, path string, max int64) ([]byte, HistoricalBytesObservation, error) {
	var zero HistoricalBytesObservation
	if !historicalBytesAbsolutePath(path) || pathWithinRoot(root, path) {
		return nil, zero, hbError("UNSAFE_PATH", "acquisition input must be absolute and outside repository")
	}
	if err := historicalBytesNoSymlinks(path); err != nil {
		return nil, zero, err
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, zero, hbWrap("INVALID_ACQUISITION", err)
	}
	if !before.Mode().IsRegular() || before.Size() > max {
		return nil, zero, hbError("INVALID_ACQUISITION", "non-regular or over-limit acquisition file")
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, zero, hbWrap("INVALID_ACQUISITION", err)
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return nil, zero, hbError("INVALID_ACQUISITION", "source changed on open")
	}
	start, err := historicalBytesStat(opened, nil)
	if err != nil {
		return nil, zero, err
	}
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(b)) > max {
		return nil, zero, hbError("SIZE_MISMATCH", "acquisition read exceeds bound")
	}
	after, err := f.Stat()
	if err != nil {
		return nil, zero, hbWrap("INVALID_ACQUISITION", err)
	}
	end, err := historicalBytesStat(after, nil)
	if err != nil {
		return nil, zero, err
	}
	last, err := os.Lstat(path)
	if err != nil || !os.SameFile(opened, last) || opened.Size() != after.Size() || opened.Size() != int64(len(b)) || start != end {
		return nil, zero, hbError("INVALID_ACQUISITION", "acquisition observation changed during read")
	}
	lastObs, err := historicalBytesStat(last, nil)
	if err != nil || lastObs != start {
		return nil, zero, hbError("INVALID_ACQUISITION", "acquisition path replaced")
	}
	obs, err := historicalBytesStat(after, b)
	return b, obs, err
}

func readHistoricalBytesStored(root string, ref RecoveryFileReference, max int64) ([]byte, error) {
	if !validRecoveryFileReference(ref) || ref.Size > max {
		return nil, hbError("CORRUPT_RECORDED_EVIDENCE", "unsafe/over-limit stored ref")
	}
	if err := checkRecoveryParents(root, ref.Path); err != nil {
		return nil, hbWrap("CORRUPT_RECORDED_EVIDENCE", err)
	}
	p := filepath.Join(root, ref.Path)
	if err := historicalBytesNoSymlinks(p); err != nil {
		return nil, hbWrap("CORRUPT_RECORDED_EVIDENCE", err)
	}
	i, err := os.Lstat(p)
	if err != nil || !i.Mode().IsRegular() || i.Size() != ref.Size {
		return nil, hbError("CORRUPT_RECORDED_EVIDENCE", "missing/nonregular stored member")
	}
	fd, err := syscall.Open(p, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, hbWrap("CORRUPT_RECORDED_EVIDENCE", err)
	}
	f := os.NewFile(uintptr(fd), p)
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(i, opened) {
		return nil, hbError("CORRUPT_RECORDED_EVIDENCE", "stored member replaced on open")
	}
	obs, err := historicalBytesStat(opened, nil)
	if err != nil {
		return nil, hbWrap("CORRUPT_RECORDED_EVIDENCE", err)
	}
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(b)) != ref.Size || sum(b) != ref.SHA256 {
		return nil, hbError("CORRUPT_RECORDED_EVIDENCE", "stored member SHA/size mismatch")
	}
	end, err := f.Stat()
	if err != nil {
		return nil, err
	}
	endObs, err := historicalBytesStat(end, nil)
	last, e := os.Lstat(p)
	if err != nil || e != nil || !os.SameFile(opened, last) || obs != endObs {
		return nil, hbError("CORRUPT_RECORDED_EVIDENCE", "stored member observation changed")
	}
	lastObs, le := historicalBytesStat(last, nil)
	if le != nil || lastObs != obs {
		return nil, hbError("CORRUPT_RECORDED_EVIDENCE", "stored path observation changed")
	}
	return b, nil
}

func readHistoricalBytesBoundedRepository(root, path string, max int64) ([]byte, error) {
	if err := checkRecoveryParents(root, path); err != nil {
		return nil, err
	}
	p := filepath.Join(root, path)
	if err := historicalBytesNoSymlinks(p); err != nil {
		return nil, err
	}
	fd, err := syscall.Open(p, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), p)
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > max {
		return nil, hbError("SIZE_MISMATCH", "stored read is nonregular/over limit")
	}
	obs, err := historicalBytesStat(info, nil)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(b)) > max || int64(len(b)) != info.Size() {
		return nil, hbError("SIZE_MISMATCH", "stored read length changed")
	}
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	end, err := historicalBytesStat(after, nil)
	last, e := os.Lstat(p)
	if err != nil || e != nil || !os.SameFile(info, last) || obs != end {
		return nil, hbError("CORRUPT_RECORDED_EVIDENCE", "stored descriptor changed")
	}
	return b, nil
}

func syncHistoricalBytesDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func publishHistoricalBytesFile(root, path string, b []byte, reuse bool) error {
	if err := checkRecoveryParents(root, path); err != nil {
		return err
	}
	ref := recoveryMemberReference(path, b)
	if _, err := os.Lstat(filepath.Join(root, path)); err == nil {
		if !reuse {
			return fmt.Errorf("write-once destination occupied: %s", path)
		}
		info, e := os.Lstat(filepath.Join(root, path))
		if e != nil || info.Mode().Perm() != 0444 {
			return fmt.Errorf("published blob must retain mode 0444: %s", path)
		}
		old, e := readHistoricalBytesStored(root, ref, int64(len(b)))
		if e != nil || !bytes.Equal(old, b) {
			return fmt.Errorf("occupied historical byte member mismatch: %s: %v", path, e)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := makeHistoricalBytesDirectories(root, filepath.Dir(filepath.Join(root, path)), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(root, path), os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Chmod(0444)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return syncHistoricalBytesDirectory(filepath.Dir(filepath.Join(root, path)))
}

// Durability includes every new directory entry, not just the final file's
// immediate parent. This runs only at publication, before the index commit.
func makeHistoricalBytesDirectories(root, dir string, mode os.FileMode) error {
	if !pathWithinRoot(root, dir) && filepath.Clean(root) != filepath.Clean(dir) {
		return hbError("UNSAFE_PATH", "publication directory escapes repository")
	}
	if err := os.MkdirAll(dir, mode); err != nil {
		return err
	}
	for p := dir; ; p = filepath.Dir(p) {
		if err := syncHistoricalBytesDirectory(p); err != nil {
			return err
		}
		if filepath.Clean(p) == filepath.Clean(root) {
			return nil
		}
		if p == filepath.Dir(p) {
			return hbError("UNSAFE_PATH", "publication root not reached")
		}
	}
}

// A sealed journal owns only its exact files and their required parent dirs.
// Unknown empty directories are evidence too; cleanup must not remove them.
func verifyHistoricalBytesDirectory(root, dir string, expected map[string]bool, pending string) error {
	if pending != "" {
		return hbError("TRANSACTION_MISMATCH", "unexpected pending inventory")
	}
	allowed := map[string]bool{dir: true}
	for p := range expected {
		for parent := filepath.ToSlash(filepath.Dir(p)); parent != dir; parent = filepath.ToSlash(filepath.Dir(parent)) {
			if parent == "." {
				return hbError("UNSAFE_PATH", "inventory escapes managed directory")
			}
			allowed[parent] = true
		}
	}
	return filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, e := repositoryRelativePath(root, path)
		if e != nil {
			return e
		}
		if d.IsDir() {
			if !allowed[rel] {
				return hbError("TRANSACTION_MISMATCH", "unknown directory in sealed inventory")
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 || !expected[rel] {
			return hbError("TRANSACTION_MISMATCH", "unknown member in sealed inventory")
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			return hbError("TRANSACTION_MISMATCH", "nonregular member in sealed inventory")
		}
		return nil
	})
}

package contentidentity

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

// This is an acceleration store, never repository authority. It is deliberately
// private to the existing invocation scope. No lookup observes repository paths:
// callers must supply their complete, freshly observed semantic input bytes.
const deterministicCacheVersion = "go-tour/deterministic-transform/v1"
const persistentResultLimit = 16 << 20
const persistentStoreLimit = 256 << 20
const persistentEntryLimit = 512

type deterministicStore struct {
	directory, implementation string
}

type deterministicEntry struct {
	Version, Implementation, Key, OutputSHA256 string
	Bytes                                      int
	Output                                     []byte `json:"-"`
}

// Only the small header is strict JSON; the checked payload stays raw bytes.
func encodeDeterministicEntry(e deterministicEntry) []byte {
	e.Bytes = len(e.Output)
	h, _ := json.Marshal(e)
	return append(append(h, '\n'), e.Output...)
}

func decodeDeterministicEntry(b []byte) (deterministicEntry, bool) {
	var e deterministicEntry
	at := bytes.IndexByte(b, '\n')
	if at < 0 || at > 1024 || strictJSONDecode(b[:at], &e) != nil {
		return e, false
	}
	e.Output = b[at+1:]
	return e, e.Bytes == len(e.Output)
}

func exactSHA256(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func invocationStore() *deterministicStore {
	dir, err := os.UserCacheDir()
	if err != nil || !filepath.IsAbs(dir) {
		return nil
	}
	// The executable binds all compiled parser/adapter/validator code and schema
	// constants. A new build is a cold namespace, never an old implementation hit.
	// Hash the running image, not its replaceable pathname. A concurrent binary
	// upgrade must not label the old process's results with the new executable's
	// identity. Platforms without this exact-image handle simply retain L1/eager.
	f, err := os.Open("/proc/self/exe")
	if err != nil {
		return nil
	}
	h := sha256.New()
	_, err = io.Copy(h, f)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return nil
	}
	return &deterministicStore{filepath.Join(dir, "go-tour-i18n", "deterministic-v1"), hex.EncodeToString(h.Sum(nil))}
}

func (s *deterministicStore) key(operation string, input []byte) string {
	b, _ := json.Marshal([]string{deterministicCacheVersion, s.implementation, operation, exactSHA256(input)})
	return exactSHA256(b)
}

// os.Root confines even a concurrent directory replacement to cache storage.
// Unsafe permissions, symlinks and all I/O failures disable this lookup/write.
func (s *deterministicStore) open(create bool) *os.Root {
	if s == nil || s.directory == "" || s.implementation == "" {
		return nil
	}
	if create {
		if err := os.MkdirAll(s.directory, 0700); err != nil {
			return nil
		}
	}
	st, err := os.Lstat(s.directory)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 {
		return nil
	}
	r, err := os.OpenRoot(s.directory)
	if err != nil {
		return nil
	}
	return r
}

func (s *deterministicStore) load(key string) ([]byte, bool) {
	r := s.open(false)
	if r == nil {
		return nil, false
	}
	defer r.Close()
	st, err := r.Lstat(key + ".json")
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 || st.Size() > persistentResultLimit+1025 {
		return nil, false
	}
	f, err := r.Open(key + ".json")
	if err != nil {
		return nil, false
	}
	b, readErr := io.ReadAll(io.LimitReader(f, persistentResultLimit+1026))
	closeErr := f.Close()
	if readErr != nil || closeErr != nil || len(b) > persistentResultLimit+1025 {
		return nil, false
	}
	e, valid := decodeDeterministicEntry(b)
	if !valid || e.Version != deterministicCacheVersion || e.Implementation != s.implementation || e.Key != key || len(e.Output) > persistentResultLimit || e.OutputSHA256 != exactSHA256(e.Output) {
		return nil, false
	}
	return e.Output, true
}

func (s *deterministicStore) save(key string, output []byte) {
	if len(output) > persistentResultLimit {
		return
	}
	r := s.open(true)
	if r == nil {
		return
	}
	defer r.Close()
	b := encodeDeterministicEntry(deterministicEntry{Version: deterministicCacheVersion, Implementation: s.implementation, Key: key, OutputSHA256: exactSHA256(output), Output: output})
	if !persistentAdmission(r, key+".json", len(b)) {
		return
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return
	}
	temporary := ".tmp-" + hex.EncodeToString(random[:])
	f, err := r.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer r.Remove(temporary)
	n, err := f.Write(b)
	if err == nil && n == len(b) {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || n != len(b) || closeErr != nil {
		return
	}
	// Atomic replacement is safe for two writers of the same pure result. An
	// interrupted or failed cache publication never fails the formal operation.
	if err := r.Rename(temporary, key+".json"); err != nil {
		return
	}
	if dir, err := r.Open("."); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
}

// A disposable acceleration store must not grow indefinitely across builds.
// This is a best-effort admission budget, not an eviction/locking subsystem:
// concurrent writers may overlap, and any inability to inspect storage simply
// declines the write. Full storage remains a cache miss, never a command error.
func persistentAdmission(root *os.Root, name string, cost int) bool {
	f, err := root.Open(".")
	if err != nil {
		return false
	}
	entries, err := f.ReadDir(persistentEntryLimit + 1)
	_ = f.Close()
	if err != nil && err != io.EOF || len(entries) > persistentEntryLimit {
		return false
	}
	if len(entries) == persistentEntryLimit {
		replacing := false
		for _, e := range entries {
			replacing = replacing || e.Name() == name
		}
		if !replacing {
			return false
		}
	}
	total := int64(cost)
	for _, e := range entries {
		if e.Name() == name {
			continue
		}
		st, err := e.Info()
		if err != nil || !st.Mode().IsRegular() {
			return false
		}
		total += st.Size()
		if total > persistentStoreLimit {
			return false
		}
	}
	return total <= persistentStoreLimit
}

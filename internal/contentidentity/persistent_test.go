package contentidentity

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func storeScope(t *testing.T, store *deterministicStore) func() {
	t.Helper()
	finish := BeginMemoization()
	activeMemo().store = store
	return finish
}

func TestPersistentTransformFallbackAndIdentity(t *testing.T) {
	s := &deterministicStore{filepath.Join(t.TempDir(), "cache"), "test-implementation"}
	input := bytes.Repeat([]byte("exact semantic inputs\n"), 256)
	want := []byte("immutable deterministic output")
	calls := 0
	run := func() {
		t.Helper()
		finish := storeScope(t, s)
		defer finish()
		got, err := Transform("expensive/v1", input, func() ([]byte, error) { calls++; return bytes.Clone(want), nil })
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("cache changed result", got, err)
		}
		got[0] ^= 1
	}
	run()
	run()
	if calls != 1 || MemoizationActive() {
		t.Fatal("no L2 hit or invocation leaked", calls)
	}
	key := s.key("expensive/v1", input)
	path := filepath.Join(s.directory, key+".json")
	valid, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, defect := range []string{"truncated", "damaged-JSON", "checksum", "version", "implementation", "key", "unknown-field", "missing"} {
		t.Run(defect, func(t *testing.T) {
			b := bytes.Clone(valid)
			e, _ := decodeDeterministicEntry(b)
			switch defect {
			case "truncated":
				b = b[:len(b)/2]
			case "damaged-JSON":
				b = []byte("not JSON")
			case "checksum":
				e.Output[0] ^= 1
				b = encodeDeterministicEntry(e)
			case "version":
				e.Version = "old"
				b = encodeDeterministicEntry(e)
			case "implementation":
				e.Implementation = "other"
				b = encodeDeterministicEntry(e)
			case "key":
				e.Key = "other"
				b = encodeDeterministicEntry(e)
			case "unknown-field":
				b = append([]byte("{\"Unknown\":true,"), b[1:]...)
			}
			if err := os.WriteFile(path, b, 0600); err != nil {
				t.Fatal(err)
			}
			if defect == "missing" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			before := calls
			run()
			if calls != before+1 {
				t.Fatal("cache damage did not recompute")
			}
		})
	}
	before := calls
	input[len(input)/2] ^= 1
	run()
	if calls != before+1 {
		t.Fatal("changed semantic input hit an old result")
	}
	if s.key("expensive/v2", input) == s.key("expensive/v1", input) || s.key("expensive/v1", input) == (&deterministicStore{s.directory, "new-build"}).key("expensive/v1", input) {
		t.Fatal("operation or executable identity missing from key")
	}
	if err := os.RemoveAll(s.directory); err != nil {
		t.Fatal(err)
	}
	before = calls
	run()
	if calls != before+1 {
		t.Fatal("deleted cache did not recompute")
	}
}

func TestPersistentUnavailableAndConcurrent(t *testing.T) {
	base := t.TempDir()
	blocked := filepath.Join(base, "file")
	if err := os.WriteFile(blocked, []byte("cannot create a directory beneath this file"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{blocked, filepath.Join(blocked, "cache")} {
		finish := storeScope(t, &deterministicStore{dir, "test"})
		for i := 0; i < 2; i++ {
			got, err := Transform("cannot-write", []byte("input"), func() ([]byte, error) { return []byte("correct"), nil })
			if err != nil || string(got) != "correct" {
				t.Fatal("unavailable cache became a command dependency", err)
			}
		}
		finish()
	}
	s := &deterministicStore{filepath.Join(base, "cache"), "test"}
	input, output := []byte("exact"), bytes.Repeat([]byte("result"), 2048)
	key := s.key("concurrent/v1", input)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.save(key, output)
			if got, ok := s.load(key); ok && !bytes.Equal(got, output) {
				t.Error("concurrent partial entry was accepted")
			}
		}()
	}
	wg.Wait()
	if got, ok := s.load(key); !ok || !bytes.Equal(got, output) {
		t.Fatal("concurrent writers did not publish complete deterministic bytes")
	}
	entries, _ := os.ReadDir(s.directory)
	if len(entries) != 1 {
		t.Fatal("temporary cache files leaked", len(entries))
	}
	if err := os.WriteFile(filepath.Join(s.directory, ".tmp-interrupted"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.load(s.key("absent", input)); ok {
		t.Fatal("unpublished bytes used")
	}
	finish := storeScope(t, s)
	defer finish()
	for i := 0; i < 2; i++ {
		_, err := Transform("failure", input, func() ([]byte, error) { return nil, errors.New("formal failure") })
		if err == nil || err.Error() != "formal failure" {
			t.Fatal("cache masked a formal failure")
		}
	}
	if _, ok := s.load(s.key("failure", input)); ok {
		t.Fatal("formal error was cached")
	}
}

func TestPersistentJSONExactValue(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	type value struct {
		Text       string
		Nil, Empty []string
		Map        map[string][]int
	}
	want := value{Text: strings.Repeat("source bytes", 900), Empty: []string{}, Map: map[string][]int{"a": nil, "b": {}}}
	b, _ := json.Marshal(want)
	for i := 0; i < 3; i++ {
		finish := BeginMemoization()
		var got value
		err := StrictJSON(b, &got)
		finish()
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatal("cold/warm parsed value differs", i, err)
		}
		got.Map["a"] = []int{1}
	}
	finish := BeginMemoization()
	defer finish()
	for _, invalid := range [][]byte{append(bytes.Clone(b[:len(b)-1]), []byte(",\"Unknown\":1}")...), append(bytes.Clone(b), 'x'), append([]byte("{\"Text\":\"duplicate\","), b[1:]...)} {
		var got value
		if StrictJSON(invalid, &got) == nil {
			t.Fatal("changed invalid source JSON reused a previous parse")
		}
	}
	var other struct{ Other string }
	if StrictJSON(b, &other) == nil {
		t.Fatal("different destination grammar reused previous parsed type")
	}
}

func TestPersistentTransformSubprocess(t *testing.T) {
	if os.Getenv("GO_TOUR_TEST_CACHE_CHILD") == "1" {
		finish := BeginMemoization()
		defer finish()
		_, err := Transform("subprocess/pure/v1", []byte(strings.Repeat("exact input", 1024)), func() ([]byte, error) {
			fmt.Println("COMPUTED")
			return []byte("exact derived bytes"), nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	if invocationStore() == nil {
		t.Skip("exact running executable image unavailable; L2 deliberately disabled")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	for pass := 0; pass < 3; pass++ {
		if pass == 2 {
			if err := os.RemoveAll(filepath.Join(os.Getenv("XDG_CACHE_HOME"), "go-tour-i18n")); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(os.Args[0], "-test.run=^TestPersistentTransformSubprocess$", "-test.count=1")
		cmd.Env = append(os.Environ(), "GO_TOUR_TEST_CACHE_CHILD=1")
		b, err := cmd.CombinedOutput()
		if err != nil || bytes.Contains(b, []byte("COMPUTED")) != (pass != 1) {
			t.Fatal("cold/warm/deleted subprocess cache behavior", pass, string(b), err)
		}
	}
}

func TestPersistentStorageBoundAndUnsafeFiles(t *testing.T) {
	s := &deterministicStore{filepath.Join(t.TempDir(), "cache"), "test"}
	if err := os.Mkdir(s.directory, 0700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < persistentEntryLimit; i++ {
		if err := os.WriteFile(filepath.Join(s.directory, fmt.Sprintf("entry-%d", i)), []byte("small"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	key := s.key("bounded", []byte("input"))
	s.save(key, []byte("output"))
	if _, err := os.Stat(filepath.Join(s.directory, key+".json")); !os.IsNotExist(err) {
		t.Fatal("full store admitted more entries")
	}
	finish := storeScope(t, s)
	got, err := Transform("bounded", []byte("input"), func() ([]byte, error) { return []byte("output"), nil })
	finish()
	if err != nil || string(got) != "output" {
		t.Fatal("full store prevented recomputation", err)
	}

	s = &deterministicStore{filepath.Join(t.TempDir(), "cache"), "test"}
	s.save(key, []byte("output"))
	path := filepath.Join(s.directory, key+".json")
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.load(key); ok {
		t.Fatal("nonprivate entry accepted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "external")
	if err := os.WriteFile(target, []byte("not authority"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.load(key); ok {
		t.Fatal("symlink entry accepted")
	}
	if err := os.Chmod(s.directory, 0755); err != nil {
		t.Fatal(err)
	}
	if r := s.open(false); r != nil {
		r.Close()
		t.Fatal("nonprivate directory accepted")
	}
}

func TestPersistentJSONMalformedDerivedValueRecomputes(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	if invocationStore() == nil {
		t.Skip("exact image unavailable")
	}
	b, _ := json.Marshal(struct{ Text string }{strings.Repeat("exact original", 1024)})
	run := func() {
		finish := BeginMemoization()
		defer finish()
		var got struct{ Text string }
		if err := StrictJSON(b, &got); err != nil || got.Text != strings.Repeat("exact original", 1024) {
			t.Fatal("damaged derived JSON became a formal error", err)
		}
	}
	run()
	s := invocationStore()
	paths, err := filepath.Glob(filepath.Join(s.directory, "*.json"))
	if err != nil || len(paths) != 1 {
		t.Fatal(paths, err)
	}
	raw, _ := os.ReadFile(paths[0])
	e, ok := decodeDeterministicEntry(raw)
	if !ok {
		t.Fatal("expected complete cache entry")
	}
	e.Output = []byte("not a parsed JSON representation")
	e.OutputSHA256 = exactSHA256(e.Output)
	if err := os.WriteFile(paths[0], encodeDeterministicEntry(e), 0600); err != nil {
		t.Fatal(err)
	}
	run()
}

func writePersistentTestEntry(t *testing.T, s *deterministicStore, input string, output []byte) string {
	t.Helper()
	if err := os.MkdirAll(s.directory, 0700); err != nil {
		t.Fatal(err)
	}
	key := s.key("admission-fixture/v1", []byte(input))
	b := encodeDeterministicEntry(deterministicEntry{Version: deterministicCacheVersion, Implementation: s.implementation, Key: key, OutputSHA256: exactSHA256(output), Output: output})
	if err := os.WriteFile(filepath.Join(s.directory, key+".json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	return key
}

func fullPersistentTestStore(t *testing.T) (*deterministicStore, string, string) {
	t.Helper()
	s := &deterministicStore{filepath.Join(t.TempDir(), "cache"), exactSHA256([]byte("current executable"))}
	current := writePersistentTestEntry(t, s, "protected", []byte("current result"))
	var oldKey string
	for i := 1; i < persistentEntryLimit; i++ {
		old := &deterministicStore{s.directory, exactSHA256([]byte(fmt.Sprintf("old executable %d", i%2)))}
		oldKey = writePersistentTestEntry(t, old, fmt.Sprint(i), []byte("old result"))
	}
	return s, current, oldKey
}

func TestPersistentStaleImplementationAdmission(t *testing.T) {
	if persistentStoreLimit != 1024<<20 || persistentEntryLimit != 2048 {
		t.Fatal("unexpected persistent budget", persistentStoreLimit, persistentEntryLimit)
	}
	s, current, old := fullPersistentTestStore(t)
	if _, ok := s.load(old); ok {
		t.Fatal("another executable's result became a current hit")
	}
	for pass := 0; pass < 2; pass++ {
		if pass == 1 {
			// Concurrent admission can leave a store one over its soft limit.
			stale := &deterministicStore{s.directory, exactSHA256([]byte("third old executable"))}
			writePersistentTestEntry(t, stale, "overlap", []byte("old overlapping result"))
		}
		key, output := s.key("new/v1", []byte(fmt.Sprint(pass))), []byte("new deterministic result")
		s.save(key, output)
		if got, ok := s.load(key); !ok || !bytes.Equal(got, output) {
			t.Fatal("stale namespaces prevented current admission", pass)
		}
		if got, ok := s.load(current); !ok || string(got) != "current result" {
			t.Fatal("current implementation entry evicted")
		}
		entries, err := os.ReadDir(s.directory)
		if err != nil || len(entries) != persistentEntryLimit {
			t.Fatal("entry budget not restored", len(entries), err)
		}
		var total int64
		for _, entry := range entries {
			st, err := entry.Info()
			if err != nil {
				t.Fatal(err)
			}
			total += st.Size()
		}
		if total > persistentStoreLimit {
			t.Fatal("byte budget exceeded", total)
		}
	}
	// Replacement of a current entry at capacity needs no eviction.
	before, _ := os.ReadDir(s.directory)
	s.save(current, []byte("current result"))
	after, _ := os.ReadDir(s.directory)
	if len(before) != len(after) {
		t.Fatal("replacement evicted unrelated entries")
	}
	for i := range before {
		if before[i].Name() != after[i].Name() {
			t.Fatal("replacement changed entry set")
		}
	}
	// Under admission pressure writers may race and decline their writes, but
	// every accepted entry stays complete and the current namespace survives.
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := s.key("pressure-concurrent/v1", []byte(fmt.Sprint(i)))
			output := []byte(fmt.Sprintf("complete result %d", i))
			s.save(key, output)
			if got, ok := s.load(key); ok && !bytes.Equal(got, output) {
				t.Error("partial concurrent entry")
			}
		}(i)
	}
	wg.Wait()
	if got, ok := s.load(current); !ok || string(got) != "current result" {
		t.Fatal("current entry evicted by concurrent admission")
	}
	entries, _ := os.ReadDir(s.directory)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tmp-") {
			t.Fatal("temporary publication leaked")
		}
	}
}

func TestPersistentCurrentNamespaceFull(t *testing.T) {
	s := &deterministicStore{filepath.Join(t.TempDir(), "cache"), "current"}
	for i := 0; i < persistentEntryLimit; i++ {
		writePersistentTestEntry(t, s, fmt.Sprint(i), []byte("current result"))
	}
	key := s.key("full-current/v1", []byte("new input"))
	s.save(key, []byte("new result"))
	if _, ok := s.load(key); ok {
		t.Fatal("current full namespace exceeded entry limit")
	}
	entries, err := os.ReadDir(s.directory)
	if err != nil || len(entries) != persistentEntryLimit {
		t.Fatal("current entries removed", len(entries), err)
	}
}

func TestPersistentStaleBytePressure(t *testing.T) {
	s := &deterministicStore{filepath.Join(t.TempDir(), "cache"), "current"}
	current := writePersistentTestEntry(t, s, "current", []byte("preserved"))
	old := &deterministicStore{s.directory, "old"}
	stale := writePersistentTestEntry(t, old, "stale", bytes.Repeat([]byte("old"), 4096))
	entries, _ := os.ReadDir(s.directory)
	var used int64
	for _, entry := range entries {
		st, _ := entry.Info()
		used += st.Size()
	}
	// Sparse, malformed ballast counts toward storage but is never eligible for
	// eviction. No test needs to materialize a GiB of payload or read this file.
	ballast := filepath.Join(s.directory, "ballast")
	f, err := os.OpenFile(ballast, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(persistentStoreLimit - used); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	key := s.key("byte-pressure/v1", []byte("new"))
	s.save(key, []byte("new"))
	if got, ok := s.load(key); !ok || string(got) != "new" {
		t.Fatal("byte pressure not reclaimed")
	}
	if _, err := os.Stat(filepath.Join(s.directory, stale+".json")); !os.IsNotExist(err) {
		t.Fatal("stale entry retained", err)
	}
	if got, ok := s.load(current); !ok || string(got) != "preserved" {
		t.Fatal("current entry lost")
	}
	entries, _ = os.ReadDir(s.directory)
	used = 0
	for _, entry := range entries {
		st, _ := entry.Info()
		used += st.Size()
	}
	if used > persistentStoreLimit || len(entries) > persistentEntryLimit {
		t.Fatal("post-eviction budget exceeded")
	}
	r := s.open(false)
	defer r.Close()
	if persistentAdmission(r, "impossible.json", persistentStoreLimit+1, s.implementation) {
		t.Fatal("oversized cost admitted")
	}
}

func TestPersistentEvictionSafetyAndFailure(t *testing.T) {
	for _, defect := range []string{"current", "malformed", "checksum", "version", "key", "permission", "symlink", "directory", "replacement", "remove-failure"} {
		t.Run(defect, func(t *testing.T) {
			s := &deterministicStore{filepath.Join(t.TempDir(), "cache"), "old"}
			key := writePersistentTestEntry(t, s, "old", []byte("exact old output"))
			name, path := key+".json", filepath.Join(s.directory, key+".json")
			b, _ := os.ReadFile(path)
			e, _ := decodeDeterministicEntry(b)
			switch defect {
			case "current":
				e.Implementation = "current"
			case "malformed":
				b = []byte("not a cache entry")
			case "checksum":
				e.Output[0] ^= 1
			case "version":
				e.Version = "unknown version"
			case "key":
				e.Key = exactSHA256([]byte("wrong key"))
			case "permission":
				if err := os.Chmod(path, 0644); err != nil {
					t.Fatal(err)
				}
			case "symlink", "directory":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if defect == "directory" {
					if err := os.Mkdir(path, 0700); err != nil {
						t.Fatal(err)
					}
				} else {
					target := filepath.Join(t.TempDir(), "outside")
					if err := os.WriteFile(target, b, 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(target, path); err != nil {
						t.Fatal(err)
					}
				}
			case "replacement":
				(&deterministicStore{s.directory, "current"}).save(key, []byte("replacement"))
			}
			if defect == "current" || defect == "checksum" || defect == "version" || defect == "key" {
				b = encodeDeterministicEntry(e)
			}
			if defect == "current" || defect == "malformed" || defect == "checksum" || defect == "version" || defect == "key" {
				if err := os.WriteFile(path, b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			r := s.open(false)
			defer r.Close()
			calls := 0
			remove := func(string) error { calls++; return errors.New("eviction denied") }
			size, err := evictStalePersistentEntry(r, name, "current", remove)
			if size != 0 || (defect == "remove-failure") != (err != nil) || (defect == "remove-failure") != (calls == 1) {
				t.Fatal("unsafe entry considered removable, or failure masked", size, calls, err)
			}
			if _, err := os.Lstat(path); err != nil {
				t.Fatal("entry removed", err)
			}
		})
	}
}

func TestPersistentEvictionFailureFallsBack(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses filesystem deletion permissions; deterministic removal failure is covered separately")
	}
	s, current, _ := fullPersistentTestStore(t)
	if err := os.Chmod(s.directory, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(s.directory, 0700)
	calls := 0
	for i := 0; i < 2; i++ {
		finish := storeScope(t, s)
		got, err := Transform("eviction-failure/v1", []byte("exact input"), func() ([]byte, error) { calls++; return []byte("canonical result"), nil })
		finish()
		if err != nil || string(got) != "canonical result" {
			t.Fatal("cache deletion failure became a formal failure", err)
		}
	}
	if calls != 2 {
		t.Fatal("read-only store unexpectedly admitted a write", calls)
	}
	if got, ok := s.load(current); !ok || string(got) != "current result" {
		t.Fatal("current entry altered")
	}
}

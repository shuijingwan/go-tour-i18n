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

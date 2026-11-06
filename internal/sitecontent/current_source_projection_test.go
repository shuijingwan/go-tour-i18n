package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Explicit readonly source-phase engineering probe; never a lifecycle command.
// Both paths observe the same disposable tree in separate transformation scopes.
func TestCurrentSourceObservationExistingTree(t *testing.T) {
	root := os.Getenv("GO_TOUR_SOURCE_OWNER_ROOT")
	if root == "" {
		t.Skip("explicit disposable source tree required")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root || !filepath.IsAbs(root) || !bytes.HasPrefix([]byte(root), []byte("/tmp/")) {
		t.Fatal("canonical disposable source tree required", err)
	}
	var canonical *CorpusSuccessor
	for _, phase := range []string{"canonical", "owned"} {
		if os.Getenv("GO_TOUR_SOURCE_OWNER_ONLY") == "owned" && phase != "owned" {
			continue
		}
		finish := contentidentity.BeginMemoization()
		v := successorValidation(root, nil)
		v.historicalMetrics = map[string]int{}
		start := time.Now()
		var got *CorpusSuccessor
		if phase == "canonical" {
			got, err = reconstructAppliedCorpusSuccessor(root, v)
		} else {
			got, err = checkedAppliedCorpusSuccessor(root, v)
		}
		elapsed := time.Since(start)
		finish()
		if err != nil || got == nil {
			t.Fatal(phase, err)
		}
		if phase == "canonical" {
			canonical = got
		} else if canonical != nil && !reflect.DeepEqual(canonical, got) {
			t.Fatal("observed source phase changed exact corpus/authority output")
		}
		b, _ := json.Marshal(map[string]any{"phase": phase, "seconds": elapsed.Seconds(), "counts": v.historicalMetrics, "identity": got.Identity})
		t.Logf("SOURCE_PHASE=%s", b)
	}
}

func TestCurrentSourceObservationOwner(t *testing.T) {
	root, _, _, parent, _ := historicalGlobalSuccessorFixture(t)
	// The unchanged eager reader is a bounded differential control, not a
	// production baseline run. Default/publication owners acquire no new node.
	eager := successorValidation(root, nil)
	eager.historicalMetrics = map[string]int{}
	startEager := time.Now()
	canonical, err := loadSourceCurrent(root, eager)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadSourceCurrent(root, eager); err != nil || eager.currentSource != nil {
		t.Fatal("default owner behavior changed", err)
	}
	eagerTime := time.Since(startEager)
	v := successorValidation(root, nil)
	v.currentSourcePhase = true
	v.historicalMetrics = map[string]int{}
	start := time.Now()
	want, err := loadSourceCurrent(root, v)
	if err != nil {
		t.Fatal(err)
	}
	first := time.Since(start)
	start = time.Now()
	got, err := loadSourceCurrent(root, v)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("same observation changed full source graph", err)
	}
	if !reflect.DeepEqual(canonical, got) || eager.historicalMetrics["current-source-projection-build"] != 2 {
		t.Fatal("changed canonical source graph or eager scheduling")
	}
	second := time.Since(start)
	if v.historicalMetrics["current-source-projection-build"] != 1 || v.historicalMetrics["current-source-projection-reuse"] != 1 {
		t.Fatal("source phase did not share derivation", v.historicalMetrics)
	}
	comparison, _ := json.Marshal(map[string]any{"canonical_two_loads_seconds": eagerTime.Seconds(), "owned_two_loads_seconds": (first + second).Seconds(), "canonical_counts": eager.historicalMetrics, "owned_counts": v.historicalMetrics, "source_identity_unchanged": canonical.Identity == got.Identity})
	t.Logf("SOURCE_OWNER_COMPARISON=%s", comparison)
	// A caller cannot alter the owner's full graph or nil/empty distinctions.
	got.Sources[0].SHA256 = digest([]byte("caller mutation"))
	got, err = loadSourceCurrent(root, v)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("caller changed retained source graph", err)
	}
	fresh := successorValidation(root, nil)
	fresh.currentSourcePhase = true
	if fresh.currentSource != nil {
		t.Fatal("another invocation inherited a graph")
	}
	if _, err := loadSourceCurrent(root, fresh); err != nil || fresh.currentSource == v.currentSource {
		t.Fatal("fresh owner not independent", err)
	}
	if _, err := v.projectCurrentSource(t.TempDir(), want); err == nil {
		t.Fatal("wrong root accepted")
	}
	wrong := *want
	wrong.Identity = digest([]byte("different historical source authority"))
	if _, err := v.projectCurrentSource(root, &wrong); err == nil {
		t.Fatal("partial/different historical basis accepted")
	}
	// Each hit freshly verifies all captured source/authority inputs, heads and
	// discovery membership. This covers a subsequent fresh record construction
	// too: its new owner must rebuild rather than inherit this owner's verdict.
	paths := map[string]string{
		"source": "_content/article.tmpl", "manifest": SourceManifestPath,
		"snapshot": GlobalPath, "corpus": i18n.GlossaryCorpusPath,
		"shell-head": ShellAdoptionCurrent, "structured-receipt": parent.Path,
	}
	for name, path := range paths {
		t.Run(name, func(t *testing.T) {
			b, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, b)
			writeFixture(t, root, path, append(bytes.Clone(b), ' '))
			if _, err := loadSourceCurrent(root, v); err == nil {
				t.Fatal("warm owner missed changed observation")
			}
			other := successorValidation(root, nil)
			if _, err := loadSourceCurrent(root, other); err == nil && (name == "source" || name == "manifest" || name == "snapshot" || name == "structured-receipt") {
				t.Fatal("second construction did not detect drift")
			}
		})
	}
	t.Run("new-discovery-head", func(t *testing.T) {
		path := PagePackageAdoptionRoot + "current/unregistered.json"
		writeFixture(t, root, path, []byte("{}\n"))
		defer os.Remove(filepath.Join(root, filepath.FromSlash(path)))
		if _, err := loadSourceCurrent(root, v); err == nil {
			t.Fatal("new authority head accepted")
		}
	})
	if got, err := loadSourceCurrent(root, v); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("restored exact observations failed", err)
	}
	failed := successorValidation(root, nil)
	failed.currentSourcePhase = true
	writeFixture(t, root, StructuredSourceReplacementPending, []byte("pending"))
	if _, err := loadSourceCurrent(root, failed); err == nil || failed.currentSource != nil {
		t.Fatal("failed construction retained a graph", err)
	}
	if err := os.Remove(filepath.Join(root, StructuredSourceReplacementPending)); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSourceCurrent(root, successorValidation(root, nil)); err != nil {
		t.Fatal("new construction reused an earlier failure", err)
	}
	t.Run("installed-corpus-phase", func(t *testing.T) {
		// The consumed-source fixture deliberately stops before its next corpus
		// adoption. Complete that existing machine CAS in this TempDir so this
		// subtest exercises a current installed projection, not a stale one.
		before, err := readRegular(root, i18n.GlossaryCorpusPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := AdoptCorpusProjection(root, digest(before), true); err != nil {
			t.Fatal(err)
		}
		control := successorValidation(root, nil)
		control.historicalMetrics = map[string]int{}
		start := time.Now()
		canonical, err := reconstructAppliedCorpusSuccessor(root, control)
		if err != nil {
			t.Fatal(err)
		}
		canonicalTime := time.Since(start)
		owned := successorValidation(root, nil)
		owned.historicalMetrics = map[string]int{}
		start = time.Now()
		result, err := checkedAppliedCorpusSuccessor(root, owned)
		if err != nil || !reflect.DeepEqual(canonical, result) {
			t.Fatal("installed corpus bytes/identity changed", err)
		}
		ownedTime := time.Since(start)
		if owned.readonlyOwner || owned.currentSourcePhase || owned.currentSource != nil {
			t.Fatal("observation owner escaped the readonly phase")
		}
		if owned.historicalMetrics["aggregate-rereads"] >= control.historicalMetrics["aggregate-rereads"] {
			t.Fatal("whole-world reconstruction still duplicated", control.historicalMetrics, owned.historicalMetrics)
		}
		if owned.historicalMetrics["manifest-file-observation"] != 1 || control.historicalMetrics["manifest-file-observation"] <= 1 {
			t.Fatal("complete manifest observation was not owned once", control.historicalMetrics, owned.historicalMetrics)
		}
		stats, _ := json.Marshal(map[string]any{"canonical_seconds": canonicalTime.Seconds(), "owned_seconds": ownedTime.Seconds(), "canonical_counts": control.historicalMetrics, "owned_counts": owned.historicalMetrics, "identity_unchanged": true})
		t.Logf("CORPUS_PHASE_COMPARISON=%s", stats)
		t.Run("complete-manifest-files-still-guarded", func(t *testing.T) {
			m, err := loadSourceManifest(root)
			if err != nil {
				t.Fatal(err)
			}
			path := m.Files[0].Path
			for _, f := range m.Files {
				if len(f.Path) > len(sourceContractRoot) && f.Path[:len(sourceContractRoot)] == sourceContractRoot {
					path = f.Path
					break
				}
			}
			b, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, b)
			_, err = owned.installedCorpusObservation(func() (*CorpusSuccessor, error) {
				r, err := reconstructAppliedCorpusSuccessor(root, owned)
				if err != nil {
					return nil, err
				}
				writeFixture(t, root, path, append(bytes.Clone(b), ' '))
				return r, nil
			})
			if err == nil {
				t.Fatal("phase accepted drift in the complete frozen manifest file set")
			}
		})
		path := "_content/article.tmpl"
		b, _ := readRegular(root, path)
		defer writeFixture(t, root, path, b)
		_, err = owned.installedCorpusObservation(func() (*CorpusSuccessor, error) {
			r, err := reconstructAppliedCorpusSuccessor(root, owned)
			if err != nil {
				return nil, err
			}
			writeFixture(t, root, path, append(bytes.Clone(b), ' '))
			return r, nil
		})
		if err == nil || owned.readonlyOwner || owned.currentSource != nil {
			t.Fatal("phase exit accepted concurrent source mutation or leaked its verdict", err)
		}
		if _, err := checkedAppliedCorpusSuccessor(root, successorValidation(root, nil)); err == nil {
			t.Fatal("second fresh construction inherited the first currentness verdict")
		}
	})
	t.Logf("SOURCE_OWNER builds=%d reuse=%d first=%s warm=%s", v.historicalMetrics["current-source-projection-build"], v.historicalMetrics["current-source-projection-reuse"], first, second)
}

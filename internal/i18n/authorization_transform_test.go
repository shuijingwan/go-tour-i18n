package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestAuthorizationPureTransformIsolation(t *testing.T) {
	key := bytes.Repeat([]byte("x"), 512)
	calls := 0
	compute := func() ([]byte, error) { calls++; return []byte{key[40]}, nil }
	done := contentidentity.BeginMemoization()
	first, err := contentidentity.Transform("authorization-test/v1", key, compute)
	if err != nil {
		t.Fatal(err)
	}
	first[0] = 'z'
	next, err := contentidentity.Transform("authorization-test/v1", key, compute)
	if err != nil || string(next) != "x" || calls != 1 {
		t.Fatal("output aliases cache or transformation was not reused", next, calls, err)
	}
	// Same short signature, different complete bytes: this must miss.
	key[40] = 'y'
	next, err = contentidentity.Transform("authorization-test/v1", key, compute)
	if err != nil || string(next) != "y" || calls != 2 {
		t.Fatal("incomplete input identity", next, calls, err)
	}
	if _, err = contentidentity.Transform("authorization-test/v2", key, compute); err != nil || calls != 3 {
		t.Fatal("operation identities conflated", calls, err)
	}
	for i := 0; i < 2; i++ {
		_, err = contentidentity.Transform("authorization-error/v1", key, func() ([]byte, error) { calls++; return nil, fmt.Errorf("failed computation") })
		if err == nil {
			t.Fatal("failure cached as success")
		}
	}
	if calls != 5 {
		t.Fatal("failure was memoized", calls)
	}
	done()
	done = contentidentity.BeginMemoization()
	defer done()
	if _, err = contentidentity.Transform("authorization-test/v1", key, compute); err != nil || calls != 6 {
		t.Fatal("cache survived invocation", calls, err)
	}
}

type authorizationCustomJSON struct{ Calls int }

var authorizationCustomReject bool

func (v *authorizationCustomJSON) UnmarshalJSON([]byte) error {
	v.Calls++
	if authorizationCustomReject {
		return fmt.Errorf("current custom decoder rejects value")
	}
	return nil
}

func TestAuthorizationTransformAndJSONCacheIsolation(t *testing.T) {
	type value struct {
		Name    string                  `json:"name"`
		Payload json.RawMessage         `json:"payload"`
		Custom  authorizationCustomJSON `json:"custom"`
	}
	raw := []byte(`{"name":"` + strings.Repeat("padding", 40) + `","payload":{"items":[1,2]},"custom":{}}`)
	var baseline value
	if err := contentidentity.StrictJSON(raw, &baseline); err != nil {
		t.Fatal(err)
	}
	done := contentidentity.BeginMemoization()
	defer done()
	if _, err := contentidentity.Transform("different-json-transform/v1", raw, func() ([]byte, error) { return []byte("unrelated output"), nil }); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		var got value
		if err := contentidentity.StrictJSON(raw, &got); err != nil || !reflect.DeepEqual(baseline, got) {
			t.Fatal("decoded values changed", err)
		}
	}
	var merged value
	for i := 0; i < 2; i++ {
		if err := contentidentity.StrictJSON(raw, &merged); err != nil {
			t.Fatal(err)
		}
	}
	if merged.Custom.Calls != 2 {
		t.Fatal("custom decoder or merge was skipped")
	}
	authorizationCustomReject = true
	defer func() { authorizationCustomReject = false }()
	if err := contentidentity.StrictJSON(raw, &value{}); err == nil {
		t.Fatal("cache reused custom decoder success")
	}
	authorizationCustomReject = false
	// A cached value for one type cannot authorize unknown members in another.
	var other struct {
		Name string `json:"name"`
	}
	if err := contentidentity.StrictJSON(raw, &other); err == nil {
		t.Fatal("JSON cache key omitted destination type")
	}
	for _, bad := range [][]byte{
		bytes.Replace(raw, []byte(`"items":[1,2]`), []byte(`"items":[1,2],"items":[]`), 1),
		bytes.Replace(raw, []byte(`"payload":`), []byte(`"unknown":`), 1),
		bytes.Replace(raw, []byte(`"name":"`+strings.Repeat("padding", 40)+`"`), []byte(`"name":null`), 1),
		append(bytes.Clone(raw), []byte(` {}`)...),
		bytes.Replace(raw, []byte("padding"), []byte{0xff}, 1),
	} {
		for i := 0; i < 2; i++ {
			if err := contentidentity.StrictJSON(bad, &value{}); err == nil {
				t.Fatal("changed malformed bytes accepted")
			}
		}
	}
}

func TestAuthorizationMemoCurrentIdentityGates(t *testing.T) {
	for _, kind := range []string{"source", "glossary", "review", "immutable reference", "validation"} {
		t.Run(kind, func(t *testing.T) {
			done := contentidentity.BeginMemoization()
			defer done()
			root, c, _, snapshot := checkpointFixture(t)
			a, path, err := EstablishCurrentTranslationAuthorization(root, c, "zh-CN", snapshot)
			if err != nil {
				t.Fatal(err)
			}
			got, err := CheckCurrentTranslationAuthorization(root, c, path)
			if err != nil || got.Identity != a.Identity {
				t.Fatal("unchanged identity", err)
			}
			var target string
			switch kind {
			case "source":
				target = c.Adapted[0].Unit.SourcePath
			case "glossary":
				target = a.Glossary.Path
			case "review":
				target = a.UnifiedReview.Path
			case "immutable reference":
				target = a.Historical.Finalization.QCResults[0].Path
			case "validation":
				target = a.Units[0].Snapshot.ValidationPath
			}
			original, err := readCompatibilityFile(root, target)
			if err != nil {
				t.Fatal(err)
			}
			writeSnapshotSource(t, root, target, append(bytes.Clone(original), '\n'))
			if _, err := CheckCurrentTranslationAuthorization(root, c, path); err == nil {
				t.Fatal("warm invocation concealed mutation", kind)
			}
			writeSnapshotSource(t, root, target, original)
			got, err = CheckCurrentTranslationAuthorization(root, c, path)
			if err != nil || got.Identity != a.Identity {
				t.Fatal("restored identity", err)
			}
		})
	}
}

func TestAuthorizationTransformBoundedFallback(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	key := bytes.Repeat([]byte("bounded"), 32)
	output := bytes.Repeat([]byte("large"), (16<<20)/5+1)
	calls := 0
	for i := 0; i < 2; i++ {
		got, err := contentidentity.Transform("large-output/v1", key, func() ([]byte, error) { calls++; return output, nil })
		if err != nil || !bytes.Equal(got, output) {
			t.Fatal("bounded fallback changed computation", err)
		}
	}
	if calls != 2 {
		t.Fatal("oversized result was cached")
	}
}

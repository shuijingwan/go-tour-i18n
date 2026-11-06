package contentidentity

import "testing"

func TestFormalArtifactPathSetAccounting(t *testing.T) {
	a, b := SHA256Hex([]byte("old")), SHA256Hex([]byte("new"))
	for _, tc := range []struct {
		name          string
		before, after map[string]string
		want          ArtifactChanges
	}{
		{"unchanged", map[string]string{"receipt": a}, map[string]string{"receipt": a}, ArtifactChanges{}},
		{"added-path-old-bytes-unchanged", map[string]string{"receipt": a}, map[string]string{"receipt": a, "pointer": a}, ArtifactChanges{PathsAdded: 1}},
		{"removed-path", map[string]string{"receipt": a, "pointer": a}, map[string]string{"receipt": a}, ArtifactChanges{PathsRemoved: 1}},
		{"changed-bytes", map[string]string{"receipt": a}, map[string]string{"receipt": b}, ArtifactChanges{BytesChanged: 1}},
		{"rename-same-bytes", map[string]string{"old": a}, map[string]string{"new": a}, ArtifactChanges{PathsAdded: 1, PathsRemoved: 1}},
		{"known-empty", map[string]string{}, map[string]string{}, ArtifactChanges{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CompareArtifactDigests(tc.before, tc.after)
			if err != nil || got != tc.want || got.Unchanged() != (tc.want == ArtifactChanges{}) {
				t.Fatalf("got %+v, %v; want %+v", got, err, tc.want)
			}
		})
	}
	for _, tc := range []struct{ before, after map[string]string }{
		{nil, map[string]string{}}, {map[string]string{}, nil},
		{map[string]string{"receipt": ""}, map[string]string{}},
	} {
		if _, err := CompareArtifactDigests(tc.before, tc.after); err == nil {
			t.Fatal("incomplete inventory accepted as unchanged")
		}
	}
}

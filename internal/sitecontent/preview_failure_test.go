package sitecontent

import (
	"bytes"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestPageAdapterFrozenPreviewProof(t *testing.T) {
	source := []byte("Read [certificate][].\n\n[certificate]: https://example.invalid/required\n")
	target := bytes.Replace(source, []byte("[certificate][]"), []byte("[证书][]"), 1)
	d, err := ParseWorkflowDocument(Source{Path: "_content/fixture.md", Kind: "page", Route: "/fixture", SHA256: digest(source)}, source, 1)
	if err != nil {
		t.Fatal(err)
	}
	var proof i18n.FrozenPreviewFailureAdapter = PageAdapter{Document: *d, Source: source}
	p := &i18n.PreviewTUFailure{Route: "/fixture", Artifact: i18n.SurfaceCandidateArtifact{Unit: i18n.QualityCheckSnapshotUnit{SourcePath: d.Path, SourceSHA256: d.SourceSHA}}, ResponseHTML: "<main>Read [证书][].</main>", MissingLinks: []string{"https://example.invalid/required"}}
	// QC/finalization/package readers use this interface with no root, no
	// completed scope, no PreviewFailureReader and no live preview dependency.
	if err := proof.VerifyFrozenPreviewTUFailure(target, p); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"source", "route", "fake-finding", "actual-link-present"} {
		t.Run(name, func(t *testing.T) {
			bad := *p
			switch name {
			case "source":
				bad.Artifact.Unit.SourceSHA256 = digest([]byte("other source"))
			case "route":
				bad.Route = "/other"
			case "fake-finding":
				bad.MissingLinks = []string{"arbitrary user text"}
			case "actual-link-present":
				bad.ResponseHTML = `<a href="https://example.invalid/required">certificate</a>`
			}
			if err := proof.VerifyFrozenPreviewTUFailure(target, &bad); err == nil {
				t.Fatal("invalid frozen proof accepted")
			}
		})
	}
}

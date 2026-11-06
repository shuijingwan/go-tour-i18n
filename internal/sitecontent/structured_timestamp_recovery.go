package sitecontent

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const StructuredTimestampRecoverySchema = "go-learning/structured-generation-timestamp-recovery/v1"

// This is an explicit metadata-loss declaration, not another invocation or a
// substitute timestamp. The maintainer attests the original session's evidence;
// hashes prove exact bytes, not the truth of a provider/session declaration.
type StructuredTimestampEvidence struct {
	Schema         string     `json:"schema"`
	Status         string     `json:"original_generated_at_status"`
	Provenance     Provenance `json:"generation"`
	BundleSHA      string     `json:"bundle_sha256"`
	BundleIdentity string     `json:"bundle_identity_sha256"`
	OutputSHA      string     `json:"output_sha256"`
	OutputZIPSHA   string     `json:"output_zip_sha256"`
	Declaration    string     `json:"declaration"`
}

type StructuredTimestampRequest struct {
	Schema     string    `json:"schema"`
	Bundle     Reference `json:"bundle"`
	Output     Reference `json:"output"`
	OutputZIP  Reference `json:"output_zip"`
	Evidence   Reference `json:"generation_session_evidence"`
	AttestedBy string    `json:"attested_by"`
}

type StructuredTimestampAuthority struct {
	Bridge     *TimestampCurrentnessBridge `json:"currentness_bridge,omitempty"`
	Schema     string                      `json:"schema"`
	Locale     string                      `json:"locale"`
	Request    Reference                   `json:"request"`
	Bundle     Reference                   `json:"bundle"`
	Output     Reference                   `json:"original_output"`
	OutputZIP  Reference                   `json:"original_output_zip"`
	Evidence   Reference                   `json:"generation_session_evidence"`
	Statement  StructuredTimestampEvidence `json:"statement"`
	AttestedBy string                      `json:"attested_by"`
	Identity   string                      `json:"identity_sha256"`
}

type StructuredTimestampRecoveryResult struct {
	Schema           string    `json:"schema"`
	Applied          bool      `json:"applied"`
	AlreadyCurrent   bool      `json:"already_current"`
	Mutation         string    `json:"mutation"`
	Authority        Reference `json:"timestamp_recovery"`
	Generation       Reference `json:"generation_evidence"`
	ValidationPassed int       `json:"validation_passed"`
}

const timestampUnavailableDeclaration = "original generated_at was never recorded and cannot be recovered from trusted evidence"

func timestampRecoveryRoot(locale string) string {
	return "data/locale-language/" + locale + "/structured-timestamp-recoveries/"
}

func timestampArchive(locale, kind, suffix string, raw []byte) Reference {
	return Reference{timestampRecoveryRoot(locale) + kind + "/" + digest(raw) + suffix, digest(raw)}
}

func validateTimestampEvidence(e StructuredTimestampEvidence) error {
	p := e.Provenance
	if e.Schema != StructuredTimestampRecoverySchema+"/generation-evidence" || e.Status != "unavailable" || e.Declaration != timestampUnavailableDeclaration || p.GeneratedAt != "" || !component(p.Session) || (p.Provider != "chatgpt" && p.Provider != "codex") || p.Model != "gpt-5.6-sol-high" || !validSHA(e.BundleSHA) || !validSHA(e.BundleIdentity) || !validSHA(e.OutputSHA) || !validSHA(e.OutputZIPSHA) {
		return fmt.Errorf("explicit unavailable original timestamp and exact Generation identity/bindings required")
	}
	return nil
}

// The original output transport has exactly one ordinary member. The JSON is
// compared byte-for-byte; neither ZIP metadata nor landing times are consulted.
func checkTimestampOutputZIP(raw, output []byte) error {
	if len(raw) > 32<<20 || len(output) > 32<<20 {
		return fmt.Errorf("timestamp recovery output transport too large")
	}
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil || len(z.File) != 1 {
		return fmt.Errorf("timestamp recovery requires one exact output ZIP member")
	}
	f := z.File[0]
	if f.Name != "structured-output.json" || !f.Mode().IsRegular() || f.UncompressedSize64 > 32<<20 {
		return fmt.Errorf("invalid timestamp recovery output ZIP member")
	}
	r, err := f.Open()
	if err != nil {
		return err
	}
	b, err := io.ReadAll(io.LimitReader(r, (32<<20)+1))
	closeErr := r.Close()
	if err != nil || closeErr != nil || !bytes.Equal(b, output) {
		return fmt.Errorf("original output ZIP/JSON bytes mismatch")
	}
	return nil
}

func timestampAuthorityRef(a StructuredTimestampAuthority) Reference {
	b, _ := Encode(a)
	return Reference{timestampRecoveryRoot(a.Locale) + "receipts/" + a.Identity + ".json", digest(b)}
}

func readTimestampInput(root string, ref Reference, v *closureValidation) ([]byte, error) {
	if !validPath(ref.Path) || !validSHA(ref.SHA256) {
		return nil, fmt.Errorf("repository-relative exact timestamp recovery reference required")
	}
	st, err := os.Lstat(pathJoin(root, ref.Path))
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > 32<<20 {
		return nil, fmt.Errorf("regular bounded timestamp recovery input required: %s", ref.Path)
	}
	var b []byte
	if v == nil {
		b, err = readRegular(root, ref.Path)
	} else {
		b, err = v.readRegular(root, ref.Path)
	}
	if err != nil {
		return nil, err
	}
	if len(b) > 32<<20 || digest(b) != ref.SHA256 {
		return nil, fmt.Errorf("timestamp recovery input SHA/size mismatch: %s", ref.Path)
	}
	return b, nil
}

// validateStructuredProvenance leaves the normal validator completely intact.
// A missing value is valid only with this exact, immutable recovery lineage.
// It cannot be combined with stale-Bundle carry-forward or an invented time.
func validateStructuredProvenance(root string, r *StructuredGeneration) ([]Reference, error) {
	if r.TimestampRecovery == nil {
		return nil, validateProvenance(r.Provenance)
	}
	if r.CarryForward != nil || r.Provenance.GeneratedAt != "" {
		return nil, fmt.Errorf("timestamp recovery cannot authorize a timestamp or carry-forward")
	}
	var a StructuredTimestampAuthority
	ab, err := readTimestampInput(root, *r.TimestampRecovery, nil)
	if err != nil {
		return nil, err
	}
	if err := StrictJSON(ab, &a); err != nil {
		return nil, err
	}
	c := a
	c.Identity = ""
	if a.Schema != StructuredTimestampRecoverySchema || a.Identity != identity(c) || timestampAuthorityRef(a) != *r.TimestampRecovery || a.Locale != r.Plan.Locale || !component(a.AttestedBy) || a.AttestedBy == r.Provenance.Session || validateTimestampEvidence(a.Statement) != nil || a.Statement.Provenance != r.Provenance || a.Bundle != r.Bundle || a.Statement.BundleSHA != r.Bundle.SHA256 || a.Statement.BundleIdentity != r.Output.BundleSHA {
		return nil, fmt.Errorf("invalid structured timestamp recovery authority/binding")
	}
	refs := []Reference{*r.TimestampRecovery, a.Request, a.Bundle, a.Output, a.OutputZIP, a.Evidence}
	data := map[Reference][]byte{}
	for _, ref := range refs[1:] {
		b, err := readTimestampInput(root, ref, nil)
		if err != nil {
			return nil, err
		}
		data[ref] = b
	}
	var q StructuredTimestampRequest
	var evidence StructuredTimestampEvidence
	if err := StrictJSON(data[a.Request], &q); err != nil {
		return nil, err
	}
	if err := StrictJSON(data[a.Evidence], &evidence); err != nil {
		return nil, err
	}
	if q.Schema != StructuredTimestampRecoverySchema+"/request" || q.AttestedBy != a.AttestedBy || q.Bundle.SHA256 != a.Bundle.SHA256 || q.Output.SHA256 != a.Output.SHA256 || q.OutputZIP.SHA256 != a.OutputZIP.SHA256 || q.Evidence.SHA256 != a.Evidence.SHA256 || !reflect.DeepEqual(evidence, a.Statement) || a.Statement.OutputSHA != a.Output.SHA256 || a.Statement.OutputZIPSHA != a.OutputZIP.SHA256 || a.Request != timestampArchive(a.Locale, "requests", ".json", data[a.Request]) || a.Evidence != timestampArchive(a.Locale, "evidence", ".json", data[a.Evidence]) || a.OutputZIP != timestampArchive(a.Locale, "output-zips", ".zip", data[a.OutputZIP]) || a.Output != structuredOutputRef(a.Locale, data[a.Output]) || a.Bundle != structuredBundleRef(a.Locale, data[a.Bundle]) {
		return nil, fmt.Errorf("timestamp recovery original evidence/archive binding mismatch")
	}
	for _, ref := range []Reference{q.Bundle, q.Output, q.OutputZIP, q.Evidence} {
		if !validPath(ref.Path) || !validSHA(ref.SHA256) {
			return nil, fmt.Errorf("invalid original timestamp recovery input reference")
		}
	}
	m, originalFiles, err := readStructuredBundle(data[a.Bundle])
	if err != nil {
		return nil, err
	}
	o, err := structuredOutputBytes(data[a.Output], *m)
	if err != nil || !reflect.DeepEqual(m.Plan, r.Plan) || !reflect.DeepEqual(o, &r.Output) {
		return nil, fmt.Errorf("timestamp recovery original Bundle/output/Plan mismatch: %v", err)
	}
	if err := checkTimestampOutputZIP(data[a.OutputZIP], data[a.Output]); err != nil {
		return nil, err
	}
	if a.Bridge != nil {
		b, err := readTimestampInput(root, a.Bridge.CurrentBundle, nil)
		if err != nil {
			return nil, err
		}
		current, fs, err := readStructuredBundle(b)
		if err != nil {
			return nil, err
		}
		if a.Bridge.CurrentBundle != structuredBundleRef(a.Locale, b) || current.Identity != a.Bridge.CurrentIdentity {
			return nil, fmt.Errorf("timestamp currentness bridge Bundle binding mismatch")
		}
		proof, dependencies, err := compatibleTimestampLineage(root, *m, originalFiles, *current, fs, false)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(proof, a.Bridge.Reattestations) {
			return nil, fmt.Errorf("timestamp currentness bridge lineage mismatch")
		}
		refs = append(refs, a.Bridge.CurrentBundle)
		refs = append(refs, dependencies...)
	}
	return refs, nil
}

// Like structured-recover, all publications are immutable content-addressed
// objects; there is no mutable selector or completion state. Default is readonly.
func RecoverStructuredTimestamp(root string, request Reference, apply bool) (*StructuredTimestampRecoveryResult, error) {
	v := newClosureValidation(root)
	var q StructuredTimestampRequest
	qb, err := readTimestampInput(root, request, v)
	if err != nil {
		return nil, err
	}
	if err := StrictJSON(qb, &q); err != nil {
		return nil, err
	}
	if q.Schema != StructuredTimestampRecoverySchema+"/request" || !component(q.AttestedBy) {
		return nil, fmt.Errorf("invalid timestamp recovery request/maintainer attestation")
	}
	inputs := map[Reference][]byte{request: qb}
	for _, ref := range []Reference{q.Bundle, q.Output, q.OutputZIP, q.Evidence} {
		b, err := readTimestampInput(root, ref, v)
		if err != nil {
			return nil, err
		}
		inputs[ref] = b
	}
	var e StructuredTimestampEvidence
	if err := StrictJSON(inputs[q.Evidence], &e); err != nil {
		return nil, err
	}
	if err := validateTimestampEvidence(e); err != nil {
		return nil, err
	}
	if q.AttestedBy == e.Provenance.Session || e.BundleSHA != q.Bundle.SHA256 || e.OutputSHA != q.Output.SHA256 || e.OutputZIPSHA != q.OutputZIP.SHA256 {
		return nil, fmt.Errorf("timestamp recovery evidence/attestation input mismatch")
	}
	if err := checkTimestampOutputZIP(inputs[q.OutputZIP], inputs[q.Output]); err != nil {
		return nil, err
	}
	old, oldFiles, err := readStructuredBundle(inputs[q.Bundle])
	if err != nil {
		return nil, err
	}
	if old.Identity != e.BundleIdentity {
		return nil, fmt.Errorf("timestamp recovery Bundle identity mismatch")
	}
	o, err := structuredOutputBytes(inputs[q.Output], *old)
	if err != nil {
		return nil, err
	}
	// Preserve the original model input. Only validated descendant shared
	// re-attestation context may bridge an ordinary comparison failure.
	bridge, currentBytes, err := timestampCurrentBundle(root, *old, inputs[q.Bundle], oldFiles)
	if err != nil {
		return nil, err
	}
	m := old
	if err := requireIndependentSession(root, m.Plan.Locale, e.Provenance.Session, "generation"); err != nil {
		return nil, err
	}
	var review i18n.UnifiedGlossaryReview
	if err := v.readReference(root, m.Plan.Review, &review); err != nil {
		return nil, err
	}
	if review.GenerationSession != e.Provenance.Session {
		return nil, fmt.Errorf("timestamp recovery original Generation session/Review mismatch")
	}
	files, err := validateStructuredOutput(root, m.Plan, *o)
	if err != nil {
		return nil, err
	}
	locale := m.Plan.Locale
	a := StructuredTimestampAuthority{Bridge: bridge, Schema: StructuredTimestampRecoverySchema, Locale: locale, Request: timestampArchive(locale, "requests", ".json", inputs[request]), Bundle: structuredBundleRef(locale, inputs[q.Bundle]), Output: structuredOutputRef(locale, inputs[q.Output]), OutputZIP: timestampArchive(locale, "output-zips", ".zip", inputs[q.OutputZIP]), Evidence: timestampArchive(locale, "evidence", ".json", inputs[q.Evidence]), Statement: e, AttestedBy: q.AttestedBy}
	a.Identity = identity(a)
	authority := timestampAuthorityRef(a)
	r := StructuredGeneration{TimestampRecovery: &authority, Schema: StructuredSchema, Plan: m.Plan, Output: *o, Provenance: e.Provenance, Bundle: a.Bundle}
	r.Identity = identity(r)
	b, err := Encode(r)
	if err != nil {
		return nil, err
	}
	result := &StructuredTimestampRecoveryResult{Schema: StructuredTimestampRecoverySchema + "/result", Mutation: "none", Authority: authority, Generation: Reference{"data/locale-language/" + locale + "/structured-generation/" + r.Identity + ".json", digest(b)}, ValidationPassed: len(files)}
	ab, _ := Encode(a)
	archives := []struct {
		ref  Reference
		data []byte
	}{{a.Request, inputs[request]}, {a.Bundle, inputs[q.Bundle]}, {a.Output, inputs[q.Output]}, {a.OutputZIP, inputs[q.OutputZIP]}, {a.Evidence, inputs[q.Evidence]}, {authority, ab}, {result.Generation, b}}
	if bridge != nil {
		last := archives[len(archives)-1]
		archives = archives[:len(archives)-1]
		archives = append(archives, struct {
			ref  Reference
			data []byte
		}{bridge.CurrentBundle, currentBytes}, last)
	}
	allPresent := true
	for _, archive := range archives {
		prior, err := v.readRegular(root, archive.ref.Path)
		if os.IsNotExist(err) {
			allPresent = false
			v.absent[archive.ref.Path] = true
			continue
		}
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(prior, archive.data) {
			return nil, fmt.Errorf("immutable timestamp recovery destination conflict: %s", archive.ref.Path)
		}
	}
	if err := v.verify(); err != nil {
		return nil, err
	}
	if allPresent {
		if _, _, _, err := readStructuredGeneration(root, result.Generation); err != nil {
			return nil, err
		}
		result.AlreadyCurrent = true
		return result, nil
	}
	if !apply {
		return result, nil
	}
	for _, archive := range archives {
		if err := SaveArtifact(root, archive.ref.Path, archive.data); err != nil {
			return nil, fmt.Errorf("timestamp recovery publication: %w; mutation=known; inspect immutable destinations before exact replay", err)
		}
	}
	if _, _, _, err := readStructuredGeneration(root, result.Generation); err != nil {
		return nil, fmt.Errorf("timestamp recovery postcommit: %w; mutation=known; stop before downstream", err)
	}
	result.Applied = true
	result.Mutation = "known"
	return result, nil
}

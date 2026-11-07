package i18n

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

type recoveryLink struct {
	Schema               string                      `json:"schema"`
	Kind                 string                      `json:"kind"`
	Locale               string                      `json:"locale"`
	RootInvocationSHA256 string                      `json:"root_invocation_sha256"`
	Prior                *TransportRecoveryReference `json:"prior,omitempty"`
	Next                 TransportRecoveryReference  `json:"next"`
	Identity             string                      `json:"identity"`
}

type recoverySealedMember struct {
	Reference RecoveryFileReference `json:"reference"`
	Data      []byte                `json:"data"`
}

type recoveryRecordTransaction struct {
	Schema    string                     `json:"schema"`
	Owner     string                     `json:"owner"`
	Root      string                     `json:"root"`
	Authority TransportRecoveryReference `json:"authority"`
	LinkPath  string                     `json:"link_path"`
	Link      []byte                     `json:"link"`
	Members   []recoverySealedMember     `json:"members"`
	Seal      string                     `json:"seal"`
}

type TransportRecoveryRecordResult struct {
	Schema    string                     `json:"schema"`
	Status    string                     `json:"status"`
	Authority TransportRecoveryReference `json:"authority"`
	Recorded  bool                       `json:"recorded"`
	Mutation  string                     `json:"mutation"`
	Pending   string                     `json:"pending,omitempty"`
}

func recoveryLinkIdentity(l recoveryLink) string            { l.Identity = ""; return sum(mustJSON(l)) }
func recoveryRecordSeal(t recoveryRecordTransaction) string { t.Seal = ""; return sum(mustJSON(t)) }
func recoveryRootDirectory(locale string) string {
	return "data/translation-transport-recoveries/" + locale
}
func recoveryRootLinkPath(a *InstalledDerivedTransportRecovery) string {
	return recoveryRootDirectory(a.Locale) + "/roots/" + a.Predecessor.Archive.SHA256 + ".json"
}
func recoveryRecordPending(a *InstalledDerivedTransportRecovery) string {
	return recoveryRootDirectory(a.Locale) + "/roots/" + a.Predecessor.Archive.SHA256 + ".record.pending"
}

func recoveryAuthorityRef(a *InstalledDerivedTransportRecovery, b []byte) TransportRecoveryReference {
	return TransportRecoveryReference{Path: transportRecoveryDirectory(a.Locale, a.Identity) + "/authority.json", SHA256: sum(b), Size: int64(len(b)), Identity: a.Identity}
}

func ReadTransportRecoveryReference(root, path, sha string) (TransportRecoveryReference, error) {
	if validateGenerationInstallPath(path) != nil || !validSHA256(sha) {
		return TransportRecoveryReference{}, fmt.Errorf("exact recovery repository path/SHA required")
	}
	b, err := readCompatibilityFile(root, path)
	if err != nil || sum(b) != sha {
		return TransportRecoveryReference{}, fmt.Errorf("recovery authority SHA mismatch: %v", err)
	}
	a, err := decodeRecoveryAuthority(b)
	if err != nil {
		return TransportRecoveryReference{}, err
	}
	ref := recoveryAuthorityRef(a, b)
	if ref.Path != path {
		return TransportRecoveryReference{}, fmt.Errorf("recovery authority location mismatch")
	}
	return ref, nil
}

func readRecoveryExactAuthority(root string, ref TransportRecoveryReference) (*InstalledDerivedTransportRecovery, map[string][]byte, error) {
	if !validRecoveryFileReference(RecoveryFileReference{ref.Path, ref.SHA256, ref.Size}) || !validSHA256(ref.Identity) {
		return nil, nil, fmt.Errorf("exact recovery authority path/SHA/size/identity required")
	}
	b, err := readCompatibilityFile(root, ref.Path)
	if err != nil || sum(b) != ref.SHA256 || int64(len(b)) != ref.Size {
		return nil, nil, fmt.Errorf("recovery authority byte reference mismatch: %v", err)
	}
	a, err := decodeRecoveryAuthority(b)
	if err != nil {
		return nil, nil, err
	}
	if a.Identity != ref.Identity || ref.Path != transportRecoveryDirectory(a.Locale, a.Identity)+"/authority.json" {
		return nil, nil, fmt.Errorf("recovery authority canonical location mismatch")
	}
	members := map[string][]byte{}
	for _, r := range a.Members {
		if _, ok := members[r.Path]; ok {
			return nil, nil, fmt.Errorf("duplicate recovery authority member")
		}
		b, err := readCompatibilityFile(root, filepath.ToSlash(filepath.Dir(ref.Path))+"/"+r.Path)
		if err != nil || sum(b) != r.SHA256 || int64(len(b)) != r.Size {
			return nil, nil, fmt.Errorf("recovery authority member integrity mismatch: %s", r.Path)
		}
		members[r.Path] = b
	}
	expected := map[string]bool{ref.Path: true}
	for _, r := range a.Members {
		expected[filepath.ToSlash(filepath.Dir(ref.Path))+"/"+r.Path] = true
	}
	if err = verifyRecoveryManagedDirectory(root, filepath.ToSlash(filepath.Dir(ref.Path)), expected, ".export.pending"); err != nil {
		return nil, nil, err
	}
	linkPath := recoveryRootLinkPath(a)
	if a.Supersedes != nil {
		linkPath = recoveryRootDirectory(a.Locale) + "/supersessions/" + a.Supersedes.Identity + ".json"
	}
	l, err := readRecoveryLink(root, linkPath)
	if err != nil || l.Next != ref || l.RootInvocationSHA256 != a.Predecessor.Archive.SHA256 || l.Locale != a.Locale {
		return nil, nil, fmt.Errorf("recovery authority has no exact committed chain link: %v", err)
	}
	if a.Supersedes != nil {
		if l.Prior == nil || *l.Prior != *a.Supersedes {
			return nil, nil, fmt.Errorf("recovery supersession link mismatch")
		}
		prior, _, err := readRecoveryExactAuthority(root, *a.Supersedes)
		if err != nil || prior.Locale != a.Locale || prior.ContentPackage.ID != a.ContentPackage.ID || prior.Predecessor.Archive != a.Predecessor.Archive {
			return nil, nil, fmt.Errorf("recovery supersession root mismatch: %v", err)
		}
	}
	return a, members, nil
}

func readRecoveryLink(root, path string) (*recoveryLink, error) {
	b, err := readCompatibilityFile(root, path)
	if err != nil {
		return nil, err
	}
	var l recoveryLink
	if err = contentidentity.StrictJSON(b, &l); err != nil {
		return nil, err
	}
	if l.Schema != "go-learning/transport-recovery-link/v1" || (l.Kind != "root" && l.Kind != "supersession") || l.Identity != recoveryLinkIdentity(l) || !validSHA256(l.RootInvocationSHA256) || ValidateLocaleName(l.Locale) != nil || (l.Kind == "root") != (l.Prior == nil) {
		return nil, fmt.Errorf("invalid recovery chain link")
	}
	return &l, nil
}

func recoveryRequireAbsent(root, path string) error {
	if validateGenerationInstallPath(path) != nil {
		return fmt.Errorf("unsafe recovery path")
	}
	if err := checkRecoveryParents(root, path); err != nil {
		return err
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); !os.IsNotExist(err) {
		return fmt.Errorf("recovery write-once path occupied: %s", path)
	}
	return nil
}

func checkRecoveryParents(root, path string) error {
	if validateGenerationInstallPath(path) != nil {
		return fmt.Errorf("unsafe recovery path")
	}
	dir := filepath.Dir(filepath.FromSlash(path))
	for dir != "." {
		info, err := os.Lstat(filepath.Join(root, dir))
		if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			return fmt.Errorf("non-directory/symlink recovery parent")
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		dir = filepath.Dir(dir)
	}
	return nil
}

func verifyRecoveryRecordPrior(root string, a *InstalledDerivedTransportRecovery) (string, error) {
	linkPath := recoveryRootLinkPath(a)
	if a.Supersedes == nil {
		if err := recoveryRequireAbsent(root, linkPath); err != nil {
			return "", err
		}
	} else {
		prior, _, err := readRecoveryExactAuthority(root, *a.Supersedes)
		if err != nil {
			return "", err
		}
		if err = recoveryRequireAbsent(root, transportRecoveryDirectory(prior.Locale, prior.Identity)+"/.export.pending"); err != nil {
			return "", err
		}
		if prior.Locale != a.Locale || prior.ContentPackage.ID != a.ContentPackage.ID || prior.Predecessor.Archive != a.Predecessor.Archive {
			return "", fmt.Errorf("supersedes wrong locale/package/root")
		}
		if bytes.Equal(mustJSON(prior.CurrentEpoch), mustJSON(a.CurrentEpoch)) && prior.Construction.SHA256 == a.Construction.SHA256 {
			return "", fmt.Errorf("current recovery leaf cannot be superseded without a new epoch")
		}
		linkPath = recoveryRootDirectory(a.Locale) + "/supersessions/" + prior.Identity + ".json"
		if err := recoveryRequireAbsent(root, linkPath); err != nil {
			return "", err
		}
		child, err := findTransportRecoveryChild(root, prior)
		if err != nil {
			return "", err
		}
		if child != nil {
			base := "data/retranslation-runs/" + a.Locale + "/" + child.BatchID + "/"
			for _, name := range []string{"raw-responses", "generation-imports", "result.json", "retries"} {
				if err := recoveryRequireAbsent(root, base+name); err != nil {
					return "", fmt.Errorf("supersedes child has installed outputs or pending")
				}
			}
		}
	}
	return linkPath, nil
}

func RecordTransportRecovery(root string, c *Catalog, o TransportRecoveryOptions, expect string, resume bool) (*TransportRecoveryRecordResult, error) {
	o.decoderContext = false
	o.contextEpoch = nil
	result := &TransportRecoveryRecordResult{Schema: "go-learning/transport-recovery-plan/v1", Status: "BLOCKED", Mutation: "none"}
	if resume {
		result.Mutation = "unknown"
	}
	if !validSHA256(expect) {
		return result, fmt.Errorf("record requires exact expect identity")
	}
	unlock, err := lockRetranslationBatchNamespace(root)
	if err != nil {
		return result, err
	}
	defer unlock()
	a, members, err := prepareTransportRecovery(root, c, o)
	if err != nil {
		return result, err
	}
	if a == nil || a.Identity != expect {
		return result, fmt.Errorf("recovery preflight CURRENT or expect/CAS identity mismatch")
	}
	linkPath := recoveryRootLinkPath(a)
	if a.Supersedes != nil {
		linkPath = recoveryRootDirectory(a.Locale) + "/supersessions/" + a.Supersedes.Identity + ".json"
	}
	pending := recoveryRecordPending(a)
	if err = checkRecoveryParents(root, pending); err != nil {
		return result, err
	}
	if !resume {
		if _, err = verifyRecoveryRecordPrior(root, a); err != nil {
			return result, err
		}
		if err = recoveryRequireAbsent(root, pending); err != nil {
			return result, err
		}
	}
	if err = os.MkdirAll(filepath.Dir(filepath.Join(root, pending)), 0755); err != nil {
		return result, err
	}
	flags := os.O_RDWR | syscall.O_NOFOLLOW
	if !resume {
		flags |= os.O_CREATE | os.O_EXCL
	}
	f, err := os.OpenFile(filepath.Join(root, pending), flags, 0600)
	if err != nil {
		return result, err
	}
	defer f.Close()
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return result, fmt.Errorf("recovery transaction has an active owner")
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	result.Pending = pending
	result.Mutation = "known"
	ab := mustJSON(a)
	ref := recoveryAuthorityRef(a, ab)
	l := recoveryLink{Schema: "go-learning/transport-recovery-link/v1", Kind: "root", Locale: a.Locale, RootInvocationSHA256: a.Predecessor.Archive.SHA256, Prior: a.Supersedes, Next: ref}
	if a.Supersedes != nil {
		l.Kind = "supersession"
	}
	l.Identity = recoveryLinkIdentity(l)
	tx := recoveryRecordTransaction{Schema: "go-learning/transport-recovery-record-transaction/v1", Owner: sum(mustJSON([]string{pending, expect})), Root: a.Predecessor.Archive.SHA256, Authority: ref, LinkPath: linkPath, Link: mustJSON(l)}
	all := map[string][]byte{"authority.json": ab}
	for k, v := range members {
		all[k] = v
	}
	var names []string
	for k := range all {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		tx.Members = append(tx.Members, recoverySealedMember{recoveryMemberReference(name, all[name]), all[name]})
	}
	tx.Seal = recoveryRecordSeal(tx)
	if resume {
		b, err := os.ReadFile(filepath.Join(root, pending))
		if err != nil {
			return result, err
		}
		var old recoveryRecordTransaction
		if contentidentity.StrictJSON(b, &old) != nil || old.Schema != tx.Schema || old.Seal != recoveryRecordSeal(old) || !bytes.Equal(mustJSON(old), mustJSON(tx)) {
			result.Mutation = "unknown"
			return result, fmt.Errorf("mutation=unknown: sealed record transaction mismatch")
		}
	} else {
		if _, err = f.Write(mustJSON(tx)); err != nil {
			return result, err
		}
		if err = f.Sync(); err != nil {
			return result, err
		}
	}
	// Fresh preflight AFTER acquiring the root owner, immediately before any
	// authority publication. The sealed plan cannot be silently refreshed.
	fresh, freshMembers, err := prepareTransportRecovery(root, c, o)
	if err != nil || fresh == nil || fresh.Identity != a.Identity || !bytes.Equal(mustJSON(freshMembers), mustJSON(members)) {
		if resume {
			result.Mutation = "unknown"
		}
		return result, fmt.Errorf("record fresh CAS mismatch: %v", err)
	}
	if committed, err := readCompatibilityFile(root, linkPath); err == nil {
		if !resume || !bytes.Equal(committed, tx.Link) {
			result.Mutation = "unknown"
			return result, fmt.Errorf("mutation=unknown: occupied record commit point")
		}
	} else {
		if !os.IsNotExist(err) {
			result.Mutation = "unknown"
			return result, err
		}
		if _, err = verifyRecoveryRecordPrior(root, a); err != nil {
			return result, err
		}
		for _, entry := range tx.Members {
			dest := transportRecoveryDirectory(a.Locale, a.Identity) + "/" + entry.Reference.Path
			if err = publishRecoverySealedFile(root, dest, entry.Data, resume); err != nil {
				result.Mutation = "unknown"
				return result, err
			}
		}
		fresh, _, err = prepareTransportRecovery(root, c, o)
		if err != nil || fresh == nil || fresh.Identity != a.Identity {
			result.Mutation = "unknown"
			return result, fmt.Errorf("record publication CAS changed: %v", err)
		}
		if _, err = verifyRecoveryRecordPrior(root, a); err != nil {
			result.Mutation = "unknown"
			return result, err
		}
		if err = publishRecoverySealedFile(root, linkPath, tx.Link, false); err != nil {
			result.Mutation = "unknown"
			return result, err
		}
	}
	if _, _, err = readRecoveryExactAuthority(root, ref); err != nil {
		result.Mutation = "unknown"
		return result, err
	}
	result.Status, result.Authority, result.Recorded = "RECORDED", ref, true
	if err = os.Remove(filepath.Join(root, pending)); err != nil {
		return result, fmt.Errorf("record committed, pending cleanup failed: %w", err)
	}
	result.Pending = ""
	return result, nil
}

func publishRecoverySealedFile(root, path string, data []byte, resume bool) error {
	if err := checkRecoveryParents(root, path); err != nil {
		return err
	}
	if b, err := readCompatibilityFile(root, path); err == nil {
		if resume && bytes.Equal(b, data) {
			return nil
		}
		return fmt.Errorf("occupied recovery publication path")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0755); err != nil {
		return err
	}
	return writeNewGenerationFile(filepath.Join(root, path), data)
}

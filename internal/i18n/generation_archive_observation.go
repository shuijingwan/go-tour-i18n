package i18n

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

type generationArchiveMember struct {
	Name         string
	Offset, Size int64
}

type generationArchiveIndex struct {
	Manifest GenerationBundleManifest
	Members  []generationArchiveMember
}

const generationArchiveIndexOperation = "generation-bundle/store-index/v1"

func generationArchiveObservationKey(data []byte) []byte {
	return mustJSON(struct {
		Contract string
		SHA256   string
		Bytes    int
	}{"generation-bundle/store-index/v1;schema=1;max-files=512;max-total-bytes=67108864", sum(data), len(data)})
}

// Callers still freshly read their archive and verify its expected import,
// provider/model, attempt, raw hashes and current dependencies. This owner is
// solely the existing invocation's pure-transformation scope. Its key is the
// complete normative SHA-256 archive identity, freshly derived from the supplied
// bytes, plus every structural decoder limit. No path or currentness verdict is
// retained. A second record build must supply and hash its fresh bytes again.
func readGenerationBundle(data []byte) (GenerationBundleManifest, map[string][]byte, error) {
	if !contentidentity.MemoizationActive() {
		return decodeGenerationBundle(data)
	}
	encoded, err := contentidentity.Transform(generationArchiveIndexOperation, generationArchiveObservationKey(data), func() ([]byte, error) {
		return indexGenerationArchive(data)
	})
	if err != nil {
		return GenerationBundleManifest{}, nil, err
	}
	var index generationArchiveIndex
	if err := decodeStrictBundleJSON(encoded, &index); err != nil {
		return decodeGenerationBundle(data)
	}
	files := make(map[string][]byte, len(index.Members))
	for _, m := range index.Members {
		if m.Offset < 0 || m.Size < 0 || m.Offset > int64(len(data))-m.Size {
			return decodeGenerationBundle(data)
		}
		// Return private buffers: downstream consumers may mutate their copy.
		files[m.Name] = bytes.Clone(data[m.Offset : m.Offset+m.Size])
	}
	return index.Manifest, files, nil
}

func indexGenerationArchive(data []byte) ([]byte, error) {
	m, files, err := decodeGenerationBundle(data)
	if err != nil {
		return nil, err
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	index := generationArchiveIndex{Manifest: m}
	for _, f := range r.File {
		offset, err := f.DataOffset()
		if err != nil {
			return nil, err
		}
		payload := files[f.Name]
		size := int64(len(payload))
		// The canonical reader already checked regular files, STORE, paths,
		// duplicates, CRC, exact inventory and expected outputs. Bind this index
		// to those exact decoded payloads; never derive it from an unchecked ZIP.
		if offset < 0 || offset > int64(len(data))-size || !bytes.Equal(payload, data[offset:offset+size]) {
			return nil, fmt.Errorf("Generation archive member is not exact stored bytes")
		}
		index.Members = append(index.Members, generationArchiveMember{f.Name, offset, size})
	}
	return json.Marshal(index)
}

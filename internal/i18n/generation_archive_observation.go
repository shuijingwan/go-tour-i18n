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
	Method       uint16
	SHA256       string
}

type generationArchiveIndex struct {
	Manifest GenerationBundleManifest
	Members  []generationArchiveMember
}

const generationArchiveIndexOperation = "generation-bundle/member-index/v2"

func generationArchiveObservationKey(data []byte) []byte {
	return mustJSON(struct {
		Contract string
		SHA256   string
		Bytes    int
	}{fmt.Sprintf("%s;schema=%d;methods=store,deflate;max-files=%d;max-total-bytes=%d", generationArchiveIndexOperation, GenerationBundleSchemaVersion, TranslationUnitGenerationBundleMaxFiles, TranslationUnitGenerationBundleMaxBytes), sum(data), len(data)})
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
	var reader *zip.Reader
	remaining := TranslationUnitGenerationBundleMaxBytes
	for i, m := range index.Members {
		if m.Size < 0 || m.Size > remaining {
			return decodeGenerationBundle(data)
		}
		switch m.Method {
		case zip.Store:
			if m.Offset < 0 || m.Offset > int64(len(data))-m.Size {
				return decodeGenerationBundle(data)
			}
			// Exact stored spans were independently verified when indexing.
			files[m.Name] = bytes.Clone(data[m.Offset : m.Offset+m.Size])
		case zip.Deflate:
			if reader == nil {
				reader, err = zip.NewReader(bytes.NewReader(data), int64(len(data)))
				if err != nil || len(reader.File) != len(index.Members) {
					return decodeGenerationBundle(data)
				}
			}
			f := reader.File[i]
			if f.Name != m.Name || f.Method != m.Method || f.UncompressedSize64 != uint64(m.Size) {
				return decodeGenerationBundle(data)
			}
			// ZIP offsets locate compressed data, never a logical payload span.
			// Decode with the same bound and CRC checks, then prove logical hash.
			payload, err := readTransportBundleFile(f, remaining)
			if err != nil {
				return GenerationBundleManifest{}, nil, err
			}
			if int64(len(payload)) != m.Size || sum(payload) != m.SHA256 {
				return GenerationBundleManifest{}, nil, fmt.Errorf("Generation archive logical member identity mismatch: %s", m.Name)
			}
			files[m.Name] = payload
		default:
			return decodeGenerationBundle(data)
		}
		remaining -= m.Size
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
		// The canonical reader already checked regular files, methods, paths,
		// duplicates, CRC, exact inventory and expected outputs. Bind this index
		// to those exact decoded payloads; never derive it from an unchecked ZIP.
		if f.Method == zip.Store && (offset < 0 || offset > int64(len(data))-size || !bytes.Equal(payload, data[offset:offset+size])) {
			return nil, fmt.Errorf("Generation archive member is not exact stored bytes")
		}
		index.Members = append(index.Members, generationArchiveMember{Name: f.Name, Offset: offset, Size: size, Method: f.Method, SHA256: sum(payload)})
	}
	return json.Marshal(index)
}

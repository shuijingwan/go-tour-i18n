package site

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

const runtimeScriptPrefix = "/_/site-runtime/"

// Match the complete frozen function, not a line number, locale or source SHA.
// Any upstream change to its control flow/assignments requires adapter review.
const frozenSetDownloadLinks = "  async function setDownloadLinks() {\n" + `    const versionElement = document.querySelector('.js-latestGoVersion');
    if (versionElement) {
      const anchorTagWindows = document.querySelector('.js-downloadWin');
      const anchorTagMac = document.querySelector('.js-downloadMac');
      const anchorTagLinux = document.querySelector('.js-downloadLinux');
      const version = await getLatestVersion();

      const macDownloadUrl = ` + "`/dl/${version}.darwin-amd64.pkg`;" + `
      const windowsDownloadUrl = ` + "`/dl/${version}.windows-amd64.msi`;" + `
      const linuxDownloadUrl = ` + "`/dl/${version}.linux-amd64.tar.gz`;" + `

      anchorTagWindows.href = windowsDownloadUrl;
      anchorTagMac.href = macDownloadUrl;
      anchorTagLinux.href = linuxDownloadUrl;

      /*
       * Note: we do not change .js-downloadBtn anymore
       * because it is impossible to tell reliably which architecture
       * the user's browser is running on.
       */
    }
  }`

const platformDownloadGuard = `      const platformAnchors = [anchorTagWindows, anchorTagMac, anchorTagLinux];
      if (platformAnchors.every(anchor => anchor === null)) return;
      if (platformAnchors.some(anchor => anchor === null)) {
        throw new Error('incomplete platform download anchors');
      }
`

// These frozen scripts contain go.dev runtime URL literals. Their derived
// delivery is locale-bound, unlike the byte-exact globally shared asset set.
// This is an explicit script allowlist, not a filesystem or upstream proxy.
func runtimeScriptSource(route string) (string, bool) {
	switch route {
	case runtimeScriptPrefix + "js/site.js":
		return "js/site.js", true
	case runtimeScriptPrefix + "doc/download.js":
		return "doc/download.js", true
	}
	return "", false
}

func runtimeScriptURL(source string) (string, bool) {
	route := runtimeScriptPrefix + source
	_, ok := runtimeScriptSource(route)
	return route, ok
}

// projectRuntimeScript applies the registered download namespace policy to
// URL literal prefixes (single/double quoted strings and template literals).
// It covers both data fetches and dynamically constructed file links without
// matching a particular line/SHA or patching global fetch. Query/interpolation
// suffixes are untouched. Frozen source and shared export bytes stay exact.
func projectRuntimeScript(r *Resolver, source string, raw []byte) ([]byte, error) {
	if _, ok := runtimeScriptURL(source); !ok {
		return nil, fmt.Errorf("unregistered runtime script: %s", source)
	}
	target, err := r.Destination("/dl/")
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(target, r.origin+"/") {
		target = strings.TrimPrefix(target, r.origin)
	}
	// Destination must remain a namespace base, not an alias to a different
	// endpoint or a URL with query/fragment. Never generate invalid JS strings.
	if target != "/dl/" && target != "https://go.dev/dl/" {
		return nil, fmt.Errorf("unsupported download runtime authority: %s", target)
	}
	result := bytes.Clone(raw)
	if source == "js/site.js" {
		if bytes.Count(result, []byte(frozenSetDownloadLinks)) != 1 || bytes.Count(result, []byte("function setDownloadLinks(")) != 1 {
			return nil, fmt.Errorf("unsupported frozen setDownloadLinks structure")
		}
		adapted := strings.Replace(frozenSetDownloadLinks, "      const version = await getLatestVersion();", platformDownloadGuard+"      const version = await getLatestVersion();", 1)
		result = bytes.Replace(result, []byte(frozenSetDownloadLinks), []byte(adapted), 1)
	}
	for _, quote := range []string{"'", `"`, "`"} {
		parts := bytes.Split(result, []byte(quote+"/dl/"))
		var out bytes.Buffer
		out.Write(parts[0])
		for _, suffix := range parts[1:] {
			base := "https://go.dev/dl/"
			// Only the JSON endpoint shares the exact /dl/ page authority.
			// Local /dl/ coverage does not authorize arbitrary file subpaths.
			if bytes.HasPrefix(suffix, []byte("?mode=json"+quote)) {
				base = target
			}
			out.WriteString(quote + base)
			out.Write(suffix)
		}
		result = out.Bytes()
	}
	return result, nil
}

// Both Preview and the sealed release handler use the same projection. The
// original script bytes come from their already validated resource inventory.
func serveRuntimeScript(w http.ResponseWriter, q *http.Request, r *Resolver, source string, raw []byte) {
	if q.Method != http.MethodGet && q.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := projectRuntimeScript(r, source, raw)
	if err != nil {
		http.Error(w, "runtime dependency policy failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if q.Method == http.MethodGet {
		_, _ = w.Write(body)
	}
}

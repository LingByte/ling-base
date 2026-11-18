package rerank

import (
	"net/http"
	"sort"
	"strings"
)

var reservedHTTPHeaders = map[string]struct{}{
	"authorization": {},
	"content-type":  {},
}

// ApplyCustomHeaders attaches user-defined HTTP headers to a request,
// skipping reserved headers (authorization, content-type).
func ApplyCustomHeaders(req *http.Request, headers map[string]string) {
	if req == nil || len(headers) == 0 {
		return
	}
	for key, value := range headers {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, reserved := reservedHTTPHeaders[strings.ToLower(key)]; reserved {
			continue
		}
		req.Header.Set(key, value)
	}
}

func newHTTPClient(cfg *RerankClientConfig) *http.Client {
	if cfg != nil && cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	timeout := DefaultTimeout
	if cfg != nil && cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	return &http.Client{Timeout: timeout}
}

func normalizeTopN(topN, docCount int) int {
	if topN <= 0 {
		topN = 5
	}
	if docCount > 0 && topN > docCount {
		topN = docCount
	}
	return topN
}

func limitResults(results []RerankResult, topN int) []RerankResult {
	if topN <= 0 || len(results) <= topN {
		return results
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results[:topN]
}

func rerankEndpoint(baseURL, suffix string) string {
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if suffix != "" && !strings.HasSuffix(endpoint, suffix) {
		endpoint += suffix
	}
	return endpoint
}

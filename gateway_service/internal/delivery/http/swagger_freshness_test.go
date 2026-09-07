package http

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

const swaggerPath = "../../../docs/swagger.json"

type swaggerDoc struct {
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

func swaggerOperations(t *testing.T) map[string]bool {
	t.Helper()

	blob, err := os.ReadFile(swaggerPath)
	if err != nil {
		t.Fatalf("đọc %s: %v", swaggerPath, err)
	}

	var doc swaggerDoc
	if err := json.Unmarshal(blob, &doc); err != nil {
		t.Fatalf("phân tích %s: %v", swaggerPath, err)
	}

	ops := make(map[string]bool)
	for path, methods := range doc.Paths {
		for method := range methods {
			ops[normalizeOperation(strings.ToUpper(method), path)] = true
		}
	}
	return ops
}

func normalizeOperation(method, path string) string {
	segments := strings.Split(path, "/")
	for i, s := range segments {
		if strings.HasPrefix(s, "{") || strings.HasPrefix(s, ":") {
			segments[i] = "{}"
		}
	}
	return method + " " + strings.Join(segments, "/")
}

func TestSwaggerJSONMatchesRoutes(t *testing.T) {
	engine := newTestEngine()
	inSwagger := swaggerOperations(t)

	var missing []string
	for _, r := range engine.Routes() {
		if !strings.HasPrefix(r.Path, "/api/") {
			continue
		}
		if !inSwagger[normalizeOperation(r.Method, r.Path)] {
			missing = append(missing, r.Method+" "+r.Path)
		}
	}
	sort.Strings(missing)

	for _, op := range missing {
		t.Errorf("swagger.json thiếu %s — chạy `make swagger`", op)
	}
	if len(missing) > 0 {
		t.Logf("thiếu %d/%d endpoint", len(missing), len(engine.Routes()))
	}
}

func TestSwaggerJSONHasNoStaleOperation(t *testing.T) {
	engine := newTestEngine()

	live := make(map[string]bool)
	for _, r := range engine.Routes() {
		live[normalizeOperation(r.Method, r.Path)] = true
	}

	var stale []string
	for op := range swaggerOperations(t) {
		if !live[op] {
			stale = append(stale, op)
		}
	}
	sort.Strings(stale)

	for _, op := range stale {
		t.Errorf("swagger.json mô tả %s nhưng route đó không còn — chạy `make swagger`", op)
	}
}

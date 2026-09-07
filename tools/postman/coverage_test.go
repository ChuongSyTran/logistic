package main

import (
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func collectionRoutes(t *testing.T) map[string]bool {
	t.Helper()

	routes := make(map[string]bool)
	var walk func(items []item)
	walk = func(items []item) {
		for _, it := range items {
			if it.Request != nil {
				routes[normalize(it.Request.Method, "/"+strings.Join(it.Request.URL.Path, "/"))] = true
			}
			walk(it.Item)
		}
	}
	walk(buildCollection().Item)
	return routes
}

func gatewayRoutes(t *testing.T) map[string]bool {
	t.Helper()

	source, err := os.ReadFile("../../gateway_service/internal/delivery/http/gateway_route.go")
	if err != nil {
		t.Fatalf("đọc bảng route: %v", err)
	}

	pattern := regexp.MustCompile(`(?m)\.(GET|POST|PUT|DELETE|PATCH)\("([^"]*)"`)
	groupPattern := regexp.MustCompile(`(?m)(\w+)\s*:?=\s*(?:secured|api|admin|publicAuth|authedAuth)\.Group\("([^"]*)"`)

	prefixes := map[string]string{
		"api":        "/api/v1",
		"secured":    "/api/v1",
		"publicAuth": "/api/v1/auth",
		"authedAuth": "/api/v1/auth",
		"admin":      "/api/v1/admin",
		"engine":     "",
	}
	for _, m := range groupPattern.FindAllStringSubmatch(string(source), -1) {
		varName, suffix := m[1], m[2]
		base := "/api/v1"
		if strings.Contains(m[0], "admin.Group") {
			base = "/api/v1/admin"
		}
		prefixes[varName] = base + suffix
	}

	routes := make(map[string]bool)
	for _, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		m := pattern.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		receiver := strings.TrimSpace(trimmed[:strings.Index(trimmed, ".")])
		prefix, known := prefixes[receiver]
		if !known {
			continue
		}
		if strings.Contains(m[2], "*") {
			continue
		}
		routes[normalize(m[1], prefix+m[2])] = true
	}
	return routes
}

func normalize(method, path string) string {
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		path = "/"
	}
	segments := strings.Split(path, "/")
	for i, s := range segments {
		if strings.HasPrefix(s, ":") || strings.HasPrefix(s, "{{") || strings.HasPrefix(s, "*") {
			segments[i] = "{}"
		}
	}
	return method + " " + strings.Join(segments, "/")
}

func TestCollectionCoversEveryRoute(t *testing.T) {
	inCollection := collectionRoutes(t)
	inGateway := gatewayRoutes(t)

	if len(inGateway) < 60 {
		t.Fatalf("chỉ trích được %d route từ gateway_route.go, có vẻ regex hỏng", len(inGateway))
	}

	var missing []string
	for route := range inGateway {
		if !inCollection[route] {
			missing = append(missing, route)
		}
	}
	sort.Strings(missing)

	for _, route := range missing {
		t.Errorf("collection thiếu route %s", route)
	}
	t.Logf("gateway có %d route, collection phủ %d", len(inGateway), len(inGateway)-len(missing))
}

func TestCollectionHasNoUnknownRoute(t *testing.T) {
	inCollection := collectionRoutes(t)
	inGateway := gatewayRoutes(t)

	allowed := map[string]bool{
		"GET /healthz":                         true,
		"GET /api/v1/vehicles/khong-phai-uuid": true,
		"GET /swagger/index.html":              true,
	}

	var unknown []string
	for route := range inCollection {
		if inGateway[route] || allowed[route] {
			continue
		}
		unknown = append(unknown, route)
	}
	sort.Strings(unknown)

	for _, route := range unknown {
		t.Errorf("collection gọi %s nhưng gateway không có route nào như vậy", route)
	}
}

func TestGeneratedFileIsUpToDate(t *testing.T) {
	const path = "../../logistic.postman_collection.json"

	existing, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("chưa sinh file collection: %v", err)
	}

	tmp := t.TempDir() + "/regenerated.json"
	cmd := exec.Command("go", "run", ".", "-o", tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("chạy lại generator: %v\n%s", err, out)
	}

	regenerated, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("đọc file vừa sinh: %v", err)
	}

	if string(existing) != string(regenerated) {
		t.Error("logistic.postman_collection.json lệch với generator — chạy `make postman` rồi commit lại")
	}
}

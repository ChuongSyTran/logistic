package http

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSwaggerAnnotationsMatchRoutes(t *testing.T) {
	engine := newTestEngine()

	realRoutes := make(map[string]string)
	for _, r := range engine.Routes() {
		if !strings.HasPrefix(r.Path, "/api/") {
			continue
		}
		realRoutes[normalizePath(r.Method, r.Path)] = r.Method + " " + r.Path
	}

	annotations, err := collectRouterAnnotations("../../controller")
	if err != nil {
		t.Fatalf("không đọc được thư mục controller: %v", err)
	}
	if len(annotations) == 0 {
		t.Fatal("không tìm thấy annotation @Router nào")
	}

	for _, a := range annotations {
		if _, ok := realRoutes[a.normalized]; !ok {
			t.Errorf("tài liệu ghi %s %s (%s:%d) nhưng KHÔNG có route nào như vậy",
				a.method, a.path, filepath.Base(a.file), a.line)
		}
	}

	t.Logf("đã đối chiếu %d annotation với %d route thật", len(annotations), len(realRoutes))
}

func TestEveryRouteIsDocumented(t *testing.T) {
	engine := newTestEngine()

	annotations, err := collectRouterAnnotations("../../controller")
	if err != nil {
		t.Fatalf("không đọc được thư mục controller: %v", err)
	}

	documented := make(map[string]bool, len(annotations))
	for _, a := range annotations {
		documented[a.normalized] = true
	}

	for _, r := range engine.Routes() {
		if !strings.HasPrefix(r.Path, "/api/") {
			continue
		}
		if !documented[normalizePath(r.Method, r.Path)] {
			t.Errorf("route %s %s chưa có annotation @Router", r.Method, r.Path)
		}
	}
}

type routerAnnotation struct {
	method     string
	path       string
	normalized string
	file       string
	line       int
}

// routerLine khớp dạng:  // @Router /api/v1/users/{id} [get]
var routerLine = regexp.MustCompile(`@Router\s+(\S+)\s+\[(\w+)\]`)

func collectRouterAnnotations(dir string) ([]routerAnnotation, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var out []routerAnnotation
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}

		full := filepath.Join(dir, e.Name())
		f, err := os.Open(full)
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(f)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			m := routerLine.FindStringSubmatch(scanner.Text())
			if m == nil {
				continue
			}
			method := strings.ToUpper(m[2])
			path := m[1]
			out = append(out, routerAnnotation{
				method:     method,
				path:       path,
				normalized: normalizePath(method, path),
				file:       full,
				line:       lineNo,
			})
		}
		_ = f.Close()

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

var (
	swaggerParam = regexp.MustCompile(`\{[^}]+\}`)
	ginParam     = regexp.MustCompile(`:[^/]+`)
	ginWildcard  = regexp.MustCompile(`\*[^/]+`)
)

func normalizePath(method, path string) string {
	p := swaggerParam.ReplaceAllString(path, "{}")
	p = ginParam.ReplaceAllString(p, "{}")
	p = ginWildcard.ReplaceAllString(p, "{}")
	p = strings.TrimSuffix(p, "/")
	return fmt.Sprintf("%s %s", strings.ToUpper(method), p)
}

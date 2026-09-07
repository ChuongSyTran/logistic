package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// Nơi ghi kết quả. Ba định dạng phục vụ ba mục đích khác nhau:
//
//	.svg     -> nhúng vào .md, đọc tài liệu là thấy hình ngay
//	.drawio  -> mở bằng diagrams.net để chỉnh tay khi cần
//	.html    -> một trang xem tất cả, không cần công cụ gì
const (
	svgDir    = "docs/diagrams/svg"
	drawioDir = "docs/diagrams"
	htmlOut   = "docs/rendered/diagrams.html"
)

// Hai loại sơ đồ, hai loại tài liệu:
//
//	sequence  -> docs/flows/     : nghiệp vụ có TRỤC THỜI GIAN
//	component -> docs/services/  : cấu trúc bên trong một service
func flowNames() map[string]bool {
	out := map[string]bool{}
	for _, s := range sequences() {
		out[s.Name] = true
	}
	return out
}

func serviceNames() map[string]bool {
	return map[string]bool{
		"gateway-service":      true,
		"auth-service":         true,
		"user-service":         true,
		"vehicle-service":      true,
		"matching-service":     true,
		"notification-service": true,
		"media-service":        true,
		"wallet-service":       true,
	}
}

// rendered gom kết quả của cả hai loại về một dạng chung để ghi file và dựng
// trang HTML.
type rendered struct {
	name    string
	title   string
	caption string
	svg     string
	drawio  string
	group   string // "arch" | "flow" | "service"
}

func renderAll() []rendered {
	var out []rendered

	for _, d := range diagrams() {
		group := "arch"
		if serviceNames()[d.Name] {
			group = "service"
		}
		out = append(out, rendered{
			name: d.Name, title: d.Title, caption: d.Caption,
			svg: renderSVG(d), drawio: renderDrawio(d), group: group,
		})
	}

	for _, s := range sequences() {
		out = append(out, rendered{
			name: s.Name, title: s.Title, caption: s.Caption,
			svg: renderSequenceSVG(s), drawio: renderSequenceDrawio(s), group: "flow",
		})
	}

	return out
}

func main() {
	all := renderAll()

	// Bắt trùng tên sớm: hai sơ đồ cùng tên sẽ ghi đè nhau trong im lặng.
	seen := map[string]bool{}
	for _, r := range all {
		if seen[r.name] {
			fmt.Fprintf(os.Stderr, "diagrams: trùng tên sơ đồ %q\n", r.name)
			os.Exit(1)
		}
		seen[r.name] = true
	}

	for _, dir := range []string{svgDir, drawioDir, filepath.Dir(htmlOut)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "diagrams: tạo thư mục %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	for _, r := range all {
		svgPath := filepath.Join(svgDir, r.name+".svg")
		if err := os.WriteFile(svgPath, []byte(r.svg), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "diagrams: ghi %s: %v\n", svgPath, err)
			os.Exit(1)
		}

		drawioPath := filepath.Join(drawioDir, r.name+".drawio")
		if err := os.WriteFile(drawioPath, []byte(r.drawio), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "diagrams: ghi %s: %v\n", drawioPath, err)
			os.Exit(1)
		}
	}

	if err := os.WriteFile(htmlOut, []byte(renderHTMLIndex(all)), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "diagrams: ghi %s: %v\n", htmlOut, err)
		os.Exit(1)
	}

	nSeq := len(sequences())
	fmt.Printf("diagrams: %d sơ đồ (%d sequence + %d thành phần) → %s/*.svg, %s/*.drawio, %s\n",
		len(all), nSeq, len(all)-nSeq, svgDir, drawioDir, htmlOut)
}

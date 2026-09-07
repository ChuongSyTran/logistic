package main

import (
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
)

// TestEdgesReferenceExistingNodes: gõ nhầm ID trong Edge là lỗi dễ mắc nhất khi
// khai báo sơ đồ bằng tay. Không có test này thì tool panic lúc chạy, hoặc tệ
// hơn là vẽ ra một mũi tên đi từ hư không.
func TestEdgesReferenceExistingNodes(t *testing.T) {
	for _, d := range diagrams() {
		ids := make(map[string]bool, len(d.Nodes))
		for _, n := range d.Nodes {
			ids[n.ID] = true
		}
		for _, e := range d.Edges {
			if !ids[e.From] {
				t.Errorf("%s: edge có From=%q nhưng không có node nào tên vậy", d.Name, e.From)
			}
			if !ids[e.To] {
				t.Errorf("%s: edge có To=%q nhưng không có node nào tên vậy", d.Name, e.To)
			}
		}
	}
}

// TestNodeIDsAreUnique: hai node trùng ID khiến map layout ghi đè, một hộp biến mất.
func TestNodeIDsAreUnique(t *testing.T) {
	for _, d := range diagrams() {
		seen := map[string]bool{}
		for _, n := range d.Nodes {
			if seen[n.ID] {
				t.Errorf("%s: trùng node ID %q", d.Name, n.ID)
			}
			seen[n.ID] = true
		}
	}
}

// TestNodesDoNotOverlap: hai hộp cùng ô lưới sẽ chồng lên nhau, đọc không ra.
// Có tính cả Span vì hộp rộng chiếm nhiều cột.
func TestNodesDoNotOverlap(t *testing.T) {
	for _, d := range diagrams() {
		occupied := map[[2]int]string{}
		for _, n := range d.Nodes {
			span := n.Span
			if span < 1 {
				span = 1
			}
			for c := n.Col; c < n.Col+span; c++ {
				cell := [2]int{c, n.Row}
				if other, taken := occupied[cell]; taken {
					t.Errorf("%s: node %q và %q cùng chiếm ô (col=%d,row=%d)",
						d.Name, other, n.ID, c, n.Row)
				}
				occupied[cell] = n.ID
			}
		}
	}
}

// TestDiagramNamesAreUnique: tên sơ đồ chính là tên file, trùng là ghi đè im lặng.
func TestDiagramNamesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range diagrams() {
		if seen[d.Name] {
			t.Errorf("trùng tên sơ đồ %q", d.Name)
		}
		seen[d.Name] = true
	}
}

// TestSVGIsWellFormedXML: SVG hỏng thì trình duyệt không hiện gì cả, và markdown
// chỉ ra một ô ảnh vỡ. Nhãn tiếng Việt có dấu nháy, "&", "<" nên phải escape đúng.
func TestSVGIsWellFormedXML(t *testing.T) {
	for _, d := range diagrams() {
		out := renderSVG(d)
		if err := xml.Unmarshal([]byte(out), new(any)); err != nil {
			t.Errorf("%s: SVG không phải XML hợp lệ: %v", d.Name, err)
		}
		if !strings.Contains(out, `fill="#FFFFFF"`) {
			t.Errorf("%s: SVG thiếu nền trắng tường minh — chữ sẽ chìm trên giao diện tối", d.Name)
		}
	}
}

// TestDrawioIsWellFormedXML: file .drawio hỏng thì diagrams.net từ chối mở.
func TestDrawioIsWellFormedXML(t *testing.T) {
	for _, d := range diagrams() {
		out := renderDrawio(d)
		if err := xml.Unmarshal([]byte(out), new(any)); err != nil {
			t.Errorf("%s: .drawio không phải XML hợp lệ: %v", d.Name, err)
		}
	}
}

// TestEscapeHandlesSpecialChars khoá lại việc escape, vì nhãn tiếng Việt trong
// spec có chứa dấu nháy kép và ký tự "&".
func TestEscapeHandlesSpecialChars(t *testing.T) {
	got := esc(`a & b < c > d "e" 'f'`)
	want := `a &amp; b &lt; c &gt; d &quot;e&quot; &#39;f&#39;`
	if got != want {
		t.Errorf("esc() = %q, mong đợi %q", got, want)
	}
}

// TestRouteIsOrthogonal: mọi đoạn của mũi tên phải song song với một trục.
// Đường chéo cắt qua các hộp khác làm sơ đồ rối.
func TestRouteIsOrthogonal(t *testing.T) {
	for _, d := range diagrams() {
		boxes, _, _ := layout(d)
		for _, e := range d.Edges {
			from, to := boxes[e.From], boxes[e.To]
			pts := route(from, to)
			if len(pts) < 2 {
				t.Errorf("%s: route %s->%s chỉ có %d điểm", d.Name, e.From, e.To, len(pts))
				continue
			}
			for i := 0; i+1 < len(pts); i++ {
				a, b := pts[i], pts[i+1]
				if a.x != b.x && a.y != b.y {
					t.Errorf("%s: route %s->%s có đoạn chéo (%d,%d)->(%d,%d)",
						d.Name, e.From, e.To, a.x, a.y, b.x, b.y)
				}
			}
		}
	}
}

// TestEveryServiceHasDiagram: mỗi tài liệu trong docs/services/ phải có đúng một
// sơ đồ thành phần cùng tên gốc.
func TestEveryServiceHasDiagram(t *testing.T) {
	have := map[string]bool{}
	for _, d := range diagrams() {
		have[d.Name] = true
	}
	for name := range serviceNames() {
		if !have[name] {
			t.Errorf("docs/services/%s.md được khai báo nhưng thiếu sơ đồ", name)
		}
	}
}

// TestCanvasSizeIsReasonable bắt trường hợp đặt nhầm Col/Row quá lớn khiến sơ đồ
// rộng hàng nghìn pixel và không ai xem nổi.
func TestCanvasSizeIsReasonable(t *testing.T) {
	const maxW, maxH = 1600, 1400
	for _, d := range diagrams() {
		_, w, h := layout(d)
		if w > maxW || h > maxH {
			t.Errorf("%s: canvas %dx%d vượt ngưỡng %dx%d — xem lại Col/Row",
				d.Name, w, h, maxW, maxH)
		}
	}
}

func TestLinesSplitsOnNewline(t *testing.T) {
	got := lines("a\nb\nc")
	if fmt.Sprint(got) != "[a b c]" {
		t.Errorf("lines() = %v", got)
	}
}

package main

import "fmt"

// Kích thước lưới. Hộp rộng hơn ô một chút thì chữ đỡ bị cắt, nên cellW > boxW.
const (
	cellW  = 260
	cellH  = 150
	boxW   = 220
	boxH   = 78
	marginX = 40
	marginY = 70 // chừa chỗ cho tiêu đề phía trên
)

// palette là bảng màu theo Kind. Cố tình chọn nền sáng + chữ đậm để đọc được
// trên cả nền trắng lẫn nền tối của GitHub — SVG sinh ra có nền trắng tường minh
// nên không phụ thuộc theme của người xem.
type colors struct{ fill, stroke, text string }

var palette = map[Kind]colors{
	KindClient:   {"#E8EAF6", "#5C6BC0", "#1A237E"},
	KindEdge:     {"#FFEBEE", "#E53935", "#B71C1C"},
	KindGateway:  {"#E3F2FD", "#1E88E5", "#0D47A1"},
	KindService:  {"#E8F5E9", "#43A047", "#1B5E20"},
	KindStore:    {"#FFF3E0", "#FB8C00", "#E65100"},
	KindBroker:   {"#F3E5F5", "#8E24AA", "#4A148C"},
	KindExternal: {"#ECEFF1", "#78909C", "#37474F"},
	KindNote:     {"#FFFDE7", "#FBC02D", "#795548"},
	KindLayer:    {"#E0F7FA", "#00ACC1", "#006064"},
}

func (k Kind) colors() colors {
	if c, ok := palette[k]; ok {
		return c
	}
	return palette[KindNote]
}

// box là hộp đã được tính toạ độ tuyệt đối.
type box struct {
	Node
	x, y, w, h int
}

func (b box) cx() int     { return b.x + b.w/2 }
func (b box) cy() int     { return b.y + b.h/2 }
func (b box) right() int  { return b.x + b.w }
func (b box) bottom() int { return b.y + b.h }

// layout tính toạ độ cho mọi node và kích thước tổng của canvas.
func layout(d Diagram) (map[string]box, int, int) {
	boxes := make(map[string]box, len(d.Nodes))
	maxCol, maxRow := 0, 0

	for _, n := range d.Nodes {
		span := n.Span
		if span < 1 {
			span = 1
		}
		w := boxW + (span-1)*cellW

		boxes[n.ID] = box{
			Node: n,
			x:    marginX + n.Col*cellW,
			y:    marginY + n.Row*cellH,
			w:    w,
			h:    boxH,
		}

		if n.Col+span-1 > maxCol {
			maxCol = n.Col + span - 1
		}
		if n.Row > maxRow {
			maxRow = n.Row
		}
	}

	width := marginX*2 + maxCol*cellW + boxW
	height := marginY + (maxRow+1)*cellH
	return boxes, width, height
}

type point struct{ x, y int }

// route tính đường đi VUÔNG GÓC giữa hai hộp, trả về danh sách điểm.
//
// Cách chọn trục: so sánh khoảng cách ngang và dọc, đi theo trục nào lớn hơn.
// Nhờ vậy mũi tên rời hộp ở cạnh gần đích nhất thay vì cắt chéo qua hộp khác.
//
// Hình dạng là chữ Z: rời hộp nguồn theo một trục, gấp khúc ở CHÍNH GIỮA, rồi
// tiến vào hộp đích cũng theo trục đó. Gấp ở giữa (chứ không gấp sát đích) giúp
// đầu mũi tên luôn cắm vuông góc vào cạnh hộp — nếu gấp sát đích thì mũi tên
// đâm ngang vào góc trên của hộp, nhìn rất khó hiểu.
func route(from, to box) []point {
	dx := to.cx() - from.cx()
	dy := to.cy() - from.cy()

	if abs(dy) >= abs(dx) {
		// Trục dọc chiếm ưu thế.
		var y1, y2 int
		if dy > 0 {
			y1, y2 = from.bottom(), to.y
		} else {
			y1, y2 = from.y, to.bottom()
		}
		x1, x2 := from.cx(), to.cx()
		if x1 == x2 {
			return []point{{x1, y1}, {x2, y2}}
		}
		mid := (y1 + y2) / 2
		return []point{{x1, y1}, {x1, mid}, {x2, mid}, {x2, y2}}
	}

	// Trục ngang chiếm ưu thế.
	var x1, x2 int
	if dx > 0 {
		x1, x2 = from.right(), to.x
	} else {
		x1, x2 = from.x, to.right()
	}
	y1, y2 := from.cy(), to.cy()
	if y1 == y2 {
		return []point{{x1, y1}, {x2, y2}}
	}
	mid := (x1 + x2) / 2
	return []point{{x1, y1}, {mid, y1}, {mid, y2}, {x2, y2}}
}

// labelAt chọn chỗ đặt nhãn: điểm giữa của đoạn dài nhất, để chữ không đè lên
// khúc gấp hay lên hộp.
func labelAt(pts []point) point {
	best, bestLen := 0, -1
	for i := 0; i+1 < len(pts); i++ {
		l := abs(pts[i+1].x-pts[i].x) + abs(pts[i+1].y-pts[i].y)
		if l > bestLen {
			best, bestLen = i, l
		}
	}
	a, b := pts[best], pts[best+1]
	return point{(a.x + b.x) / 2, (a.y + b.y) / 2}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func esc(s string) string {
	out := ""
	for _, r := range s {
		switch r {
		case '&':
			out += "&amp;"
		case '<':
			out += "&lt;"
		case '>':
			out += "&gt;"
		case '"':
			out += "&quot;"
		case '\'':
			out += "&#39;"
		default:
			out += string(r)
		}
	}
	return out
}

func lines(label string) []string {
	var out []string
	cur := ""
	for _, r := range label {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	return append(out, cur)
}

func must(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

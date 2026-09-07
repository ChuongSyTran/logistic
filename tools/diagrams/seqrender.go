package main

import (
	"fmt"
	"strings"
)

// Kích thước sequence diagram.
const (
	seqLaneW    = 200 // khoảng cách giữa hai lifeline
	seqHeadW    = 168 // bề ngang hộp tên
	seqHeadH    = 52
	seqHeadY    = 76
	seqFirstMsg = 56  // khoảng cách từ đáy hộp tên tới message đầu
	seqMsgGap   = 52  // khoảng cách dọc giữa hai message
	seqSelfGap  = 34  // chiều cao thêm cho message tự gọi
	seqMarginX  = 48
	seqActW     = 12 // bề ngang thanh activation
	seqFragPad  = 26 // lề của khung alt/opt so với lifeline ngoài cùng
)

type seqGeom struct {
	x      map[string]int // tâm mỗi lifeline
	y      []int          // toạ độ dọc của từng message
	width  int
	height int
	order  map[string]int // thứ tự cột, để biết trái/phải
}

// seqLayout tính toạ độ. Message tự gọi chiếm nhiều chiều cao hơn vì phải vẽ
// một vòng cung sang phải rồi quay lại.
func seqLayout(s Sequence) seqGeom {
	g := seqGeom{
		x:     make(map[string]int, len(s.Lifelines)),
		order: make(map[string]int, len(s.Lifelines)),
		y:     make([]int, len(s.Messages)),
	}

	for i, l := range s.Lifelines {
		g.x[l.ID] = seqMarginX + seqHeadW/2 + i*seqLaneW
		g.order[l.ID] = i
	}

	cur := seqHeadY + seqHeadH + seqFirstMsg
	for i, m := range s.Messages {
		g.y[i] = cur
		cur += seqMsgGap
		if m.Kind == Self {
			cur += seqSelfGap
		}
		if m.Note != "" {
			cur += 16
		}
	}

	g.width = seqMarginX*2 + seqHeadW + (len(s.Lifelines)-1)*seqLaneW
	g.height = cur + 40
	return g
}

// activation tính các thanh activation trên mỗi lifeline.
//
// Quy tắc: message Sync đi TỚI một lifeline thì mở activation ở đó; message
// Return đi RA khỏi lifeline đó thì đóng lại. Async không mở activation vì bên
// gửi không chờ. Thanh nào chưa đóng tới cuối sơ đồ thì đóng ở message cuối.
type actBar struct {
	lifeline string
	y1, y2   int
	depth    int // lồng nhau: thanh trong dịch sang phải một chút
}

func activations(s Sequence, g seqGeom) []actBar {
	type open struct{ y, depth int }
	stacks := map[string][]open{}
	var bars []actBar

	for i, m := range s.Messages {
		y := g.y[i]

		switch m.Kind {
		case Sync:
			if m.From == m.To {
				continue
			}
			d := len(stacks[m.To])
			stacks[m.To] = append(stacks[m.To], open{y: y, depth: d})

		case Return:
			st := stacks[m.From]
			if len(st) == 0 {
				continue
			}
			top := st[len(st)-1]
			stacks[m.From] = st[:len(st)-1]
			bars = append(bars, actBar{lifeline: m.From, y1: top.y, y2: y, depth: top.depth})
		}
	}

	// Đóng nốt các thanh còn treo.
	lastY := g.height - 60
	if len(g.y) > 0 {
		lastY = g.y[len(g.y)-1] + 18
	}
	for id, st := range stacks {
		for _, o := range st {
			bars = append(bars, actBar{lifeline: id, y1: o.y, y2: lastY, depth: o.depth})
		}
	}
	return bars
}

// fragBox tính hình chữ nhật của một khung alt/opt.
type fragBox struct {
	Fragment
	x1, y1, x2, y2 int
	elseY          int
}

func fragBoxes(s Sequence, g seqGeom) []fragBox {
	var out []fragBox
	for _, f := range s.Fragments {
		if f.From < 0 || f.To >= len(s.Messages) || f.From > f.To {
			continue
		}

		minX, maxX := 1<<30, 0
		for i := f.From; i <= f.To; i++ {
			m := s.Messages[i]
			for _, id := range []string{m.From, m.To} {
				if x, ok := g.x[id]; ok {
					if x < minX {
						minX = x
					}
					if x > maxX {
						maxX = x
					}
				}
			}
		}

		fb := fragBox{
			Fragment: f,
			x1:       minX - seqFragPad - 40,
			x2:       maxX + seqFragPad + 40,
			y1:       g.y[f.From] - 34,
			y2:       g.y[f.To] + 22,
			elseY:    -1,
		}
		if f.Else >= 0 && f.Else > f.From && f.Else <= f.To {
			fb.elseY = g.y[f.Else] - 24
		}
		out = append(out, fb)
	}
	return out
}

// ---------------------------------------------------------------------------
// SVG
// ---------------------------------------------------------------------------

func renderSequenceSVG(s Sequence) string {
	g := seqLayout(s)
	bars := activations(s, g)
	frags := fragBoxes(s, g)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="Segoe UI, Roboto, Helvetica, Arial, sans-serif">`,
		g.width, g.height, g.width, g.height)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#FFFFFF"/>`, g.width, g.height)

	// Hai kiểu đầu mũi tên theo chuẩn UML: đặc cho lời gọi, mảnh cho trả về / async.
	b.WriteString(`<defs>`)
	b.WriteString(`<marker id="solid" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" fill="#37474F"/></marker>`)
	b.WriteString(`<marker id="open" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="9" markerHeight="9" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10" fill="none" stroke="#37474F" stroke-width="1.4"/></marker>`)
	b.WriteString(`</defs>`)

	fmt.Fprintf(&b, `<text x="%d" y="32" font-size="19" font-weight="600" fill="#263238">%s</text>`,
		seqMarginX, esc(s.Title))
	if s.Caption != "" {
		fmt.Fprintf(&b, `<text x="%d" y="54" font-size="12.5" fill="#607D8B">%s</text>`,
			seqMarginX, esc(s.Caption))
	}

	// 1. Khung alt/opt vẽ trước, nằm dưới cùng.
	for _, f := range frags {
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" rx="4" fill="#FAFAFA" fill-opacity="0.55" stroke="#90A4AE" stroke-width="1.2"/>`,
			f.x1, f.y1, f.x2-f.x1, f.y2-f.y1)
		// Tab góc trên trái ghi loại khung.
		fmt.Fprintf(&b, `<path d="M %d %d h 46 l 12 12 v 8 h -58 z" fill="#ECEFF1" stroke="#90A4AE" stroke-width="1.2"/>`,
			f.x1, f.y1)
		fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="11" font-weight="700" fill="#37474F">%s</text>`,
			f.x1+8, f.y1+14, esc(f.Type))
		fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="11" fill="#455A64">[%s]</text>`,
			f.x1+66, f.y1+14, esc(f.Label))

		if f.elseY > 0 {
			fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#90A4AE" stroke-width="1.1" stroke-dasharray="5 4"/>`,
				f.x1, f.elseY, f.x2, f.elseY)
			fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="11" fill="#455A64">[%s]</text>`,
				f.x1+8, f.elseY+14, esc(f.ElseLabel))
		}
	}

	// 2. Lifeline (đường đứt dọc).
	for _, l := range s.Lifelines {
		x := g.x[l.ID]
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#B0BEC5" stroke-width="1.2" stroke-dasharray="6 5"/>`,
			x, seqHeadY+seqHeadH, x, g.height-24)
	}

	// 3. Thanh activation.
	for _, bar := range bars {
		x := g.x[bar.lifeline] - seqActW/2 + bar.depth*5
		h := bar.y2 - bar.y1
		if h < 12 {
			h = 12
		}
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" fill="#CFD8DC" stroke="#78909C" stroke-width="1"/>`,
			x, bar.y1, seqActW, h)
	}

	// 4. Message.
	for i, m := range s.Messages {
		y := g.y[i]
		writeSeqMessage(&b, m, i+1, g, y)
	}

	// 5. Hộp tên lifeline vẽ sau cùng để đè lên đầu đường kẻ.
	for _, l := range s.Lifelines {
		x := g.x[l.ID]
		c := l.Kind.colors()
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" rx="7" fill="%s" stroke="%s" stroke-width="1.6"/>`,
			x-seqHeadW/2, seqHeadY, seqHeadW, seqHeadH, c.fill, c.stroke)

		ls := lines(l.Label)
		startY := seqHeadY + seqHeadH/2 - (len(ls)-1)*8 + 4
		for j, line := range ls {
			size, weight := 12.0, "400"
			if j == 0 {
				size, weight = 12.8, "600"
			}
			fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="%.1f" font-weight="%s" fill="%s" text-anchor="middle">%s</text>`,
				x, startY+j*15, size, weight, c.text, esc(line))
		}
	}

	b.WriteString(`</svg>`)
	return b.String()
}

func writeSeqMessage(b *strings.Builder, m Message, num int, g seqGeom, y int) {
	x1, ok1 := g.x[m.From]
	x2, ok2 := g.x[m.To]
	if !ok1 || !ok2 {
		return
	}

	stroke := "#37474F"
	dash := ""
	marker := "solid"

	switch m.Kind {
	case Return:
		dash = ` stroke-dasharray="7 4"`
		marker = "open"
		stroke = "#546E7A"
	case Async:
		marker = "open"
	}

	label := fmt.Sprintf("%d. %s", num, m.Label)

	if m.Kind == Self || m.From == m.To {
		// Vòng cung sang phải rồi quay lại cùng lifeline.
		const w, h = 46, 26
		fmt.Fprintf(b, `<polyline points="%d,%d %d,%d %d,%d %d,%d" fill="none" stroke="%s" stroke-width="1.5" marker-end="url(#solid)"/>`,
			x1+seqActW/2, y, x1+seqActW/2+w, y, x1+seqActW/2+w, y+h, x1+seqActW/2+6, y+h, stroke)
		fmt.Fprintf(b, `<text x="%d" y="%d" font-size="11.5" fill="#263238">%s</text>`,
			x1+seqActW/2+w+10, y+6, esc(label))
		if m.Note != "" {
			fmt.Fprintf(b, `<text x="%d" y="%d" font-size="10.5" font-style="italic" fill="#78909C">%s</text>`,
				x1+seqActW/2+w+10, y+21, esc(m.Note))
		}
		return
	}

	// Bắt đầu/kết thúc sát mép thanh activation thay vì tâm lifeline.
	from, to := x1, x2
	if x2 > x1 {
		from += seqActW / 2
		to -= seqActW / 2
	} else {
		from -= seqActW / 2
		to += seqActW / 2
	}

	fmt.Fprintf(b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1.5"%s marker-end="url(#%s)"/>`,
		from, y, to, y, stroke, dash, marker)

	mid := (from + to) / 2
	fmt.Fprintf(b, `<text x="%d" y="%d" font-size="11.5" fill="#263238" text-anchor="middle">%s</text>`,
		mid, y-7, esc(label))
	if m.Note != "" {
		fmt.Fprintf(b, `<text x="%d" y="%d" font-size="10.5" font-style="italic" fill="#78909C" text-anchor="middle">%s</text>`,
			mid, y+15, esc(m.Note))
	}
}

// ---------------------------------------------------------------------------
// DRAWIO
// ---------------------------------------------------------------------------

// renderSequenceDrawio sinh mxGraph XML cho sequence diagram.
//
// Draw.io không có "kiểu sequence" tự động, nên mọi thứ được vẽ bằng shape có
// toạ độ tuyệt đối: hộp tên, đường lifeline, thanh activation, mũi tên có
// sourcePoint/targetPoint cố định. Đổi lại, file mở ra là chỉnh tay được ngay.
func renderSequenceDrawio(s Sequence) string {
	g := seqLayout(s)
	bars := activations(s, g)
	frags := fragBoxes(s, g)

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<mxfile host="app.diagrams.net" type="device">` + "\n")
	fmt.Fprintf(&b, `  <diagram name="%s">`+"\n", esc(s.Title))
	fmt.Fprintf(&b, `    <mxGraphModel dx="%d" dy="%d" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" pageWidth="%d" pageHeight="%d" math="0" shadow="0">`+"\n",
		g.width, g.height, g.width, g.height)
	b.WriteString("      <root>\n        <mxCell id=\"0\" />\n        <mxCell id=\"1\" parent=\"0\" />\n")

	fmt.Fprintf(&b, `        <mxCell id="title" value="%s" style="text;html=1;fontSize=19;fontStyle=1;fontColor=#263238;" vertex="1" parent="1"><mxGeometry x="%d" y="12" width="%d" height="26" as="geometry"/></mxCell>`+"\n",
		esc(s.Title), seqMarginX, g.width-seqMarginX*2)
	if s.Caption != "" {
		fmt.Fprintf(&b, `        <mxCell id="caption" value="%s" style="text;html=1;fontSize=12;fontColor=#607D8B;" vertex="1" parent="1"><mxGeometry x="%d" y="38" width="%d" height="20" as="geometry"/></mxCell>`+"\n",
			esc(s.Caption), seqMarginX, g.width-seqMarginX*2)
	}

	// Khung alt/opt.
	for i, f := range frags {
		title := f.Type + " [" + f.Label + "]"
		fmt.Fprintf(&b, `        <mxCell id="frag%d" value="%s" style="shape=umlFrame;whiteSpace=wrap;html=1;width=110;height=24;fillColor=none;strokeColor=#90A4AE;fontSize=11;fontColor=#37474F;align=left;verticalAlign=top;spacingLeft=6;" vertex="1" parent="1"><mxGeometry x="%d" y="%d" width="%d" height="%d" as="geometry"/></mxCell>`+"\n",
			i, esc(title), f.x1, f.y1, f.x2-f.x1, f.y2-f.y1)

		if f.elseY > 0 {
			fmt.Fprintf(&b, `        <mxCell id="fragelse%d" value="[%s]" style="text;html=1;fontSize=11;fontColor=#455A64;align=left;" vertex="1" parent="1"><mxGeometry x="%d" y="%d" width="%d" height="18" as="geometry"/></mxCell>`+"\n",
				i, esc(f.ElseLabel), f.x1+8, f.elseY, f.x2-f.x1-16)
			fmt.Fprintf(&b, `        <mxCell id="fragline%d" style="endArrow=none;html=1;dashed=1;strokeColor=#90A4AE;" edge="1" parent="1"><mxGeometry relative="1" as="geometry"><mxPoint x="%d" y="%d" as="sourcePoint"/><mxPoint x="%d" y="%d" as="targetPoint"/></mxGeometry></mxCell>`+"\n",
				i, f.x1, f.elseY, f.x2, f.elseY)
		}
	}

	// Hộp tên + đường lifeline.
	for i, l := range s.Lifelines {
		x := g.x[l.ID]
		c := l.Kind.colors()
		value := strings.ReplaceAll(esc(l.Label), "\n", "&#10;")

		fmt.Fprintf(&b, `        <mxCell id="ll%d" value="%s" style="rounded=1;arcSize=12;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;fontColor=%s;fontSize=12;fontStyle=1;" vertex="1" parent="1"><mxGeometry x="%d" y="%d" width="%d" height="%d" as="geometry"/></mxCell>`+"\n",
			i, value, c.fill, c.stroke, c.text, x-seqHeadW/2, seqHeadY, seqHeadW, seqHeadH)

		fmt.Fprintf(&b, `        <mxCell id="lline%d" style="endArrow=none;html=1;dashed=1;strokeColor=#B0BEC5;" edge="1" parent="1"><mxGeometry relative="1" as="geometry"><mxPoint x="%d" y="%d" as="sourcePoint"/><mxPoint x="%d" y="%d" as="targetPoint"/></mxGeometry></mxCell>`+"\n",
			i, x, seqHeadY+seqHeadH, x, g.height-24)
	}

	// Thanh activation.
	for i, bar := range bars {
		x := g.x[bar.lifeline] - seqActW/2 + bar.depth*5
		h := bar.y2 - bar.y1
		if h < 12 {
			h = 12
		}
		fmt.Fprintf(&b, `        <mxCell id="act%d" value="" style="rounded=0;html=1;fillColor=#CFD8DC;strokeColor=#78909C;" vertex="1" parent="1"><mxGeometry x="%d" y="%d" width="%d" height="%d" as="geometry"/></mxCell>`+"\n",
			i, x, bar.y1, seqActW, h)
	}

	// Message.
	for i, m := range s.Messages {
		y := g.y[i]
		x1, ok1 := g.x[m.From]
		x2, ok2 := g.x[m.To]
		if !ok1 || !ok2 {
			continue
		}

		label := esc(fmt.Sprintf("%d. %s", i+1, m.Label))

		if m.Kind == Self || m.From == m.To {
			fmt.Fprintf(&b, `        <mxCell id="m%d" value="%s" style="edgeStyle=orthogonalEdgeStyle;html=1;rounded=0;endArrow=block;endFill=1;strokeColor=#37474F;fontSize=11;align=left;verticalAlign=middle;labelPosition=right;spacingLeft=6;" edge="1" parent="1"><mxGeometry relative="1" as="geometry"><mxPoint x="%d" y="%d" as="sourcePoint"/><mxPoint x="%d" y="%d" as="targetPoint"/><Array as="points"><mxPoint x="%d" y="%d"/><mxPoint x="%d" y="%d"/></Array></mxGeometry></mxCell>`+"\n",
				i, label, x1+seqActW/2, y, x1+seqActW/2+6, y+26, x1+seqActW/2+46, y, x1+seqActW/2+46, y+26)
			continue
		}

		style := "html=1;rounded=0;endArrow=block;endFill=1;strokeColor=#37474F;fontSize=11;verticalAlign=bottom;"
		switch m.Kind {
		case Return:
			style = "html=1;rounded=0;endArrow=open;endFill=0;dashed=1;strokeColor=#546E7A;fontSize=11;verticalAlign=bottom;"
		case Async:
			style = "html=1;rounded=0;endArrow=open;endFill=0;strokeColor=#37474F;fontSize=11;verticalAlign=bottom;"
		}

		from, to := x1, x2
		if x2 > x1 {
			from += seqActW / 2
			to -= seqActW / 2
		} else {
			from -= seqActW / 2
			to += seqActW / 2
		}

		fmt.Fprintf(&b, `        <mxCell id="m%d" value="%s" style="%s" edge="1" parent="1"><mxGeometry relative="1" as="geometry"><mxPoint x="%d" y="%d" as="sourcePoint"/><mxPoint x="%d" y="%d" as="targetPoint"/></mxGeometry></mxCell>`+"\n",
			i, label, style, from, y, to, y)

		if m.Note != "" {
			fmt.Fprintf(&b, `        <mxCell id="n%d" value="%s" style="text;html=1;fontSize=10;fontStyle=2;fontColor=#78909C;align=center;" vertex="1" parent="1"><mxGeometry x="%d" y="%d" width="%d" height="16" as="geometry"/></mxCell>`+"\n",
				i, esc(m.Note), min(from, to), y+4, abs(to-from))
		}
	}

	b.WriteString("      </root>\n    </mxGraphModel>\n  </diagram>\n</mxfile>\n")
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

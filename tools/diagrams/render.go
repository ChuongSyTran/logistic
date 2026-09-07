package main

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// SVG — nhúng thẳng vào markdown, GitHub và IDE đều render được
// ---------------------------------------------------------------------------

// renderSVG vẽ sơ đồ ra SVG tự chứa (không phụ thuộc font hay CSS bên ngoài).
//
// Nền được tô trắng TƯỜNG MINH: GitHub hiển thị SVG trong thẻ <img>, nếu để nền
// trong suốt thì trên giao diện tối chữ đen sẽ chìm hoàn toàn.
func renderSVG(d Diagram) string {
	boxes, w, h := layout(d)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="Segoe UI, Roboto, Helvetica, Arial, sans-serif">`,
		w, h, w, h)

	// Nền trắng + định nghĩa đầu mũi tên.
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#FFFFFF"/>`, w, h)
	b.WriteString(`<defs>`)
	b.WriteString(`<marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" fill="#546E7A"/></marker>`)
	b.WriteString(`</defs>`)

	// Tiêu đề.
	fmt.Fprintf(&b, `<text x="%d" y="30" font-size="19" font-weight="600" fill="#263238">%s</text>`,
		marginX, esc(d.Title))
	if d.Caption != "" {
		fmt.Fprintf(&b, `<text x="%d" y="52" font-size="12.5" fill="#607D8B">%s</text>`,
			marginX, esc(d.Caption))
	}

	// Vẽ mũi tên trước để hộp nằm đè lên, che phần đường thừa.
	for _, e := range d.Edges {
		from, okF := boxes[e.From]
		to, okT := boxes[e.To]
		must(okF, "diagram %s: edge trỏ tới node không tồn tại: %s", d.Name, e.From)
		must(okT, "diagram %s: edge trỏ tới node không tồn tại: %s", d.Name, e.To)

		pts := route(from, to)
		dash := ""
		if e.Dashed {
			dash = ` stroke-dasharray="6 4"`
		}

		coords := make([]string, 0, len(pts))
		for _, p := range pts {
			coords = append(coords, fmt.Sprintf("%d,%d", p.x, p.y))
		}
		fmt.Fprintf(&b, `<polyline points="%s" fill="none" stroke="#546E7A" stroke-width="1.6"%s marker-end="url(#arrow)"/>`,
			strings.Join(coords, " "), dash)

		if e.Label != "" {
			at := labelAt(pts)
			// Nền trắng sau chữ để đường kẻ không xuyên qua chữ.
			width := len([]rune(e.Label))*6 + 10
			fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="17" rx="3" fill="#FFFFFF" opacity="0.92"/>`,
				at.x-width/2, at.y-9, width)
			fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="11" fill="#455A64" text-anchor="middle">%s</text>`,
				at.x, at.y+3, esc(e.Label))
		}
	}

	// Hộp.
	for _, n := range d.Nodes {
		bx := boxes[n.ID]
		c := n.Kind.colors()

		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" rx="8" fill="%s" stroke="%s" stroke-width="1.6"/>`,
			bx.x, bx.y, bx.w, bx.h, c.fill, c.stroke)

		ls := lines(n.Label)
		startY := bx.cy() - (len(ls)-1)*8 + 4
		for i, line := range ls {
			size, weight := 12.0, "400"
			if i == 0 {
				size, weight = 13.0, "600"
			}
			fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="%.1f" font-weight="%s" fill="%s" text-anchor="middle">%s</text>`,
				bx.cx(), startY+i*16, size, weight, c.text, esc(line))
		}
	}

	b.WriteString(`</svg>`)
	return b.String()
}

// ---------------------------------------------------------------------------
// DRAWIO — file nguồn mở được bằng diagrams.net để chỉnh tay
// ---------------------------------------------------------------------------

// renderDrawio sinh mxGraph XML ở dạng KHÔNG nén, nhờ vậy file đọc được bằng mắt
// và `git diff` có nghĩa. Draw.io mặc định nén base64+deflate, khi đó mỗi lần
// sửa một chữ là cả file đổi trắng.
func renderDrawio(d Diagram) string {
	boxes, w, h := layout(d)

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&b, `<mxfile host="app.diagrams.net" type="device">`+"\n")
	fmt.Fprintf(&b, `  <diagram name="%s">`+"\n", esc(d.Title))
	fmt.Fprintf(&b, `    <mxGraphModel dx="%d" dy="%d" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" pageWidth="%d" pageHeight="%d" math="0" shadow="0">`+"\n",
		w, h, w, h)
	b.WriteString(`      <root>` + "\n")
	b.WriteString(`        <mxCell id="0" />` + "\n")
	b.WriteString(`        <mxCell id="1" parent="0" />` + "\n")

	// Tiêu đề và chú thích dưới dạng text cell.
	fmt.Fprintf(&b, `        <mxCell id="title" value="%s" style="text;html=1;fontSize=19;fontStyle=1;fontColor=#263238;verticalAlign=middle;" vertex="1" parent="1"><mxGeometry x="%d" y="8" width="%d" height="26" as="geometry"/></mxCell>`+"\n",
		esc(d.Title), marginX, w-marginX*2)
	if d.Caption != "" {
		fmt.Fprintf(&b, `        <mxCell id="caption" value="%s" style="text;html=1;fontSize=12;fontColor=#607D8B;verticalAlign=middle;" vertex="1" parent="1"><mxGeometry x="%d" y="34" width="%d" height="20" as="geometry"/></mxCell>`+"\n",
			esc(d.Caption), marginX, w-marginX*2)
	}

	for _, n := range d.Nodes {
		bx := boxes[n.ID]
		c := n.Kind.colors()
		// Draw.io hiển thị "\n" trong value nếu dùng thực thể &#10; kèm whiteSpace=wrap.
		value := strings.ReplaceAll(esc(n.Label), "\n", "&#10;")

		fmt.Fprintf(&b, `        <mxCell id="%s" value="%s" style="rounded=1;arcSize=10;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;fontColor=%s;fontSize=12;align=center;verticalAlign=middle;" vertex="1" parent="1"><mxGeometry x="%d" y="%d" width="%d" height="%d" as="geometry"/></mxCell>`+"\n",
			esc(n.ID), value, c.fill, c.stroke, c.text, bx.x, bx.y, bx.w, bx.h)
	}

	for i, e := range d.Edges {
		dash := "0"
		if e.Dashed {
			dash = "1"
		}
		fmt.Fprintf(&b, `        <mxCell id="e%d" value="%s" style="edgeStyle=orthogonalEdgeStyle;rounded=1;html=1;strokeColor=#546E7A;strokeWidth=1.6;dashed=%s;fontSize=11;fontColor=#455A64;endArrow=block;endFill=1;" edge="1" parent="1" source="%s" target="%s"><mxGeometry relative="1" as="geometry"/></mxCell>`+"\n",
			i, esc(e.Label), dash, esc(e.From), esc(e.To))
	}

	b.WriteString(`      </root>` + "\n")
	b.WriteString(`    </mxGraphModel>` + "\n")
	b.WriteString(`  </diagram>` + "\n")
	b.WriteString(`</mxfile>` + "\n")
	return b.String()
}

// ---------------------------------------------------------------------------
// HTML — một trang duy nhất xem được toàn bộ sơ đồ
// ---------------------------------------------------------------------------

func renderHTMLIndex(ds []rendered) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="vi">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Sơ đồ kiến trúc — Logistics OS</title>
<style>
  :root { color-scheme: light; }
  body { margin:0; background:#F5F7F9; color:#263238;
         font-family:"Segoe UI",Roboto,Helvetica,Arial,sans-serif; }
  header { background:#263238; color:#fff; padding:24px 32px; }
  header h1 { margin:0 0 6px; font-size:22px; }
  header p  { margin:0; opacity:.75; font-size:13.5px; }
  nav { padding:16px 32px; background:#fff; border-bottom:1px solid #DDE3E7;
        position:sticky; top:0; z-index:10; }
  nav a { display:inline-block; margin:0 14px 8px 0; font-size:13px;
          color:#1E88E5; text-decoration:none; }
  nav a:hover { text-decoration:underline; }
  nav strong { display:block; margin:10px 0 6px; font-size:12px;
               text-transform:uppercase; letter-spacing:.06em; color:#78909C; }
  section { margin:28px 32px; background:#fff; border:1px solid #DDE3E7;
            border-radius:10px; padding:20px; }
  section h2 { margin:0 0 4px; font-size:17px; }
  section p.cap { margin:0 0 16px; font-size:13px; color:#607D8B; }
  .scroll { overflow-x:auto; }
  svg { display:block; }
  footer { margin:40px 32px; font-size:12.5px; color:#78909C; }
  table.legend { border-collapse:collapse; font-size:13px; }
  table.legend td { padding:5px 12px 5px 0; vertical-align:middle; }
  code { background:#ECEFF1; padding:1px 5px; border-radius:3px; font-size:12px; }
</style>
</head>
<body>
<header>
  <h1>Sơ đồ kiến trúc — Logistics OS</h1>
  <p>Trang này do <code>make diagrams</code> sinh ra. Đừng sửa tay: sửa spec trong <code>tools/diagrams/</code> rồi chạy lại.</p>
</header>
<nav>
`)

	writeNav := func(title, group string) {
		fmt.Fprintf(&b, "  <strong>%s</strong>\n", title)
		for _, d := range ds {
			if d.group == group {
				fmt.Fprintf(&b, `  <a href="#%s">%s</a>`+"\n", d.name, esc(d.title))
			}
		}
	}
	writeNav("Kiến trúc", "arch")
	writeNav("Luồng nghiệp vụ — sequence diagram", "flow")
	writeNav("Service — sơ đồ thành phần", "service")

	b.WriteString("</nav>\n")

	// Chú giải ký hiệu sequence, đặt ngay đầu trang để người đọc không phải đoán.
	b.WriteString(`<section id="legend">
  <h2>Chú giải ký hiệu</h2>
  <p class="cap">Áp dụng cho các sequence diagram bên dưới.</p>
  <table class="legend">
    <tr><td><svg width="90" height="16"><line x1="4" y1="8" x2="72" y2="8" stroke="#37474F" stroke-width="1.5"/><path d="M72 4 L82 8 L72 12 z" fill="#37474F"/></svg></td>
        <td><strong>Gọi đồng bộ</strong> — bên gửi chờ kết quả</td></tr>
    <tr><td><svg width="90" height="16"><line x1="4" y1="8" x2="72" y2="8" stroke="#546E7A" stroke-width="1.5" stroke-dasharray="7 4"/><path d="M72 4 L82 8 L72 12" fill="none" stroke="#546E7A" stroke-width="1.4"/></svg></td>
        <td><strong>Giá trị trả về</strong></td></tr>
    <tr><td><svg width="90" height="16"><line x1="4" y1="8" x2="72" y2="8" stroke="#37474F" stroke-width="1.5"/><path d="M72 4 L82 8 L72 12" fill="none" stroke="#37474F" stroke-width="1.4"/></svg></td>
        <td><strong>Bất đồng bộ</strong> — phát rồi đi tiếp, không chờ</td></tr>
    <tr><td><svg width="90" height="26"><polyline points="4,6 40,6 40,20 10,20" fill="none" stroke="#37474F" stroke-width="1.5"/><path d="M14 16 L4 20 L14 24" fill="#37474F"/></svg></td>
        <td><strong>Tự gọi</strong> — tính toán hoặc kiểm tra nội bộ</td></tr>
    <tr><td><svg width="90" height="26"><rect x="38" y="2" width="12" height="22" fill="#CFD8DC" stroke="#78909C"/></svg></td>
        <td><strong>Thanh activation</strong> — khoảng thời gian đối tượng đang xử lý</td></tr>
    <tr><td><svg width="90" height="26"><rect x="4" y="2" width="82" height="22" fill="#FAFAFA" stroke="#90A4AE"/><path d="M4 2 h30 l8 8 v6 h-38 z" fill="#ECEFF1" stroke="#90A4AE"/><text x="8" y="12" font-size="9" font-weight="700" fill="#37474F">alt</text></svg></td>
        <td><strong>Khung alt / opt</strong> — nhánh rẽ, đường đứt bên trong ngăn nhánh else</td></tr>
  </table>
</section>
`)

	for _, d := range ds {
		fmt.Fprintf(&b, `<section id="%s">`+"\n", d.name)
		fmt.Fprintf(&b, `  <h2>%s</h2>`+"\n", esc(d.title))
		if d.caption != "" {
			fmt.Fprintf(&b, `  <p class="cap">%s</p>`+"\n", esc(d.caption))
		}
		b.WriteString(`  <div class="scroll">` + "\n  ")
		b.WriteString(d.svg)
		b.WriteString("\n  </div>\n</section>\n")
	}

	b.WriteString(`<footer>Nguồn: <code>tools/diagrams/</code> · Sinh lại: <code>make diagrams</code></footer>
</body>
</html>
`)
	return b.String()
}

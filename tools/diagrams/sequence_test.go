package main

import (
	"encoding/xml"
	"strings"
	"testing"
)

// TestSequenceMessagesReferenceLifelines: gõ nhầm ID lifeline trong Message là
// lỗi hay gặp nhất khi khai báo sequence bằng tay — mũi tên sẽ biến mất im lặng.
func TestSequenceMessagesReferenceLifelines(t *testing.T) {
	for _, s := range sequences() {
		ids := map[string]bool{}
		for _, l := range s.Lifelines {
			ids[l.ID] = true
		}
		for i, m := range s.Messages {
			if !ids[m.From] {
				t.Errorf("%s: message %d có From=%q không phải lifeline", s.Name, i, m.From)
			}
			if !ids[m.To] {
				t.Errorf("%s: message %d có To=%q không phải lifeline", s.Name, i, m.To)
			}
		}
	}
}

func TestSequenceLifelineIDsAreUnique(t *testing.T) {
	for _, s := range sequences() {
		seen := map[string]bool{}
		for _, l := range s.Lifelines {
			if seen[l.ID] {
				t.Errorf("%s: trùng lifeline ID %q", s.Name, l.ID)
			}
			seen[l.ID] = true
		}
	}
}

// TestFragmentRangesAreValid: chỉ số message sai khiến khung alt vẽ lệch chỗ
// hoặc bao trùm nhầm nhóm message — người đọc hiểu sai nhánh rẽ.
func TestFragmentRangesAreValid(t *testing.T) {
	for _, s := range sequences() {
		n := len(s.Messages)
		for _, f := range s.Fragments {
			if f.From < 0 || f.To >= n {
				t.Errorf("%s: fragment %q có dải [%d..%d] ngoài phạm vi 0..%d",
					s.Name, f.Label, f.From, f.To, n-1)
				continue
			}
			if f.From > f.To {
				t.Errorf("%s: fragment %q có From > To (%d > %d)", s.Name, f.Label, f.From, f.To)
			}
			if f.Else >= 0 {
				if f.Else <= f.From || f.Else > f.To {
					t.Errorf("%s: fragment %q có Else=%d nằm ngoài (%d..%d]",
						s.Name, f.Label, f.Else, f.From, f.To)
				}
				if f.ElseLabel == "" {
					t.Errorf("%s: fragment %q có nhánh else nhưng thiếu ElseLabel", s.Name, f.Label)
				}
			}
		}
	}
}

// TestReturnMessagesHaveMatchingCall: một mũi tên trả về mà trước đó không có
// lời gọi đồng bộ nào tới lifeline đó là dấu hiệu sai thứ tự — activation sẽ
// không khớp và sơ đồ đọc ra sai nghĩa.
//
// Điểm tinh tế: trong khung alt chỉ MỘT nhánh thực sự chạy. Hai nhánh cùng trả
// về từ một lifeline là hợp lệ (ví dụ "422 lỗi" và "200 thành công" cùng đáp lại
// một lời gọi). Vì vậy phải kiểm từng ĐƯỜNG CHẠY riêng biệt, chứ đọc tuần tự cả
// danh sách message sẽ báo sai.
func TestReturnMessagesHaveMatchingCall(t *testing.T) {
	for _, s := range sequences() {
		for pathIdx, path := range branchPaths(s) {
			depth := map[string]int{}
			for _, i := range path {
				m := s.Messages[i]
				switch m.Kind {
				case Sync:
					if m.From != m.To {
						depth[m.To]++
					}
				case Return:
					if depth[m.From] == 0 {
						t.Errorf("%s (đường chạy #%d): message %d trả về từ %q nhưng chưa có lời gọi đồng bộ nào tới đó",
							s.Name, pathIdx, i, m.From)
						continue
					}
					depth[m.From]--
				}
			}
		}
	}
}

// branchPaths liệt kê mọi ĐƯỜNG CHẠY khả dĩ của sequence: với mỗi khung alt có
// nhánh else, chọn một trong hai nhánh và loại bỏ nhánh kia.
//
// Số đường chạy là 2^(số fragment có else) — các sequence ở đây có nhiều nhất
// 3 fragment nên tối đa 8 đường, hoàn toàn kiểm hết được.
func branchPaths(s Sequence) [][]int {
	var alts []Fragment
	for _, f := range s.Fragments {
		if f.Else >= 0 {
			alts = append(alts, f)
		}
	}

	total := 1 << len(alts)
	paths := make([][]int, 0, total)

	for combo := 0; combo < total; combo++ {
		excluded := map[int]bool{}
		for k, f := range alts {
			takeFirstBranch := combo&(1<<k) == 0
			for i := f.From; i <= f.To; i++ {
				inElse := i >= f.Else
				// Loại bỏ nhánh KHÔNG được chọn.
				if (takeFirstBranch && inElse) || (!takeFirstBranch && !inElse) {
					excluded[i] = true
				}
			}
		}

		path := make([]int, 0, len(s.Messages))
		for i := range s.Messages {
			if !excluded[i] {
				path = append(path, i)
			}
		}
		paths = append(paths, path)
	}

	return paths
}

func TestSequenceSVGIsWellFormed(t *testing.T) {
	for _, s := range sequences() {
		out := renderSequenceSVG(s)
		if err := xml.Unmarshal([]byte(out), new(any)); err != nil {
			t.Errorf("%s: SVG không hợp lệ: %v", s.Name, err)
		}
		if !strings.Contains(out, `fill="#FFFFFF"`) {
			t.Errorf("%s: thiếu nền trắng tường minh", s.Name)
		}
		// Phải có đủ hai kiểu đầu mũi tên theo chuẩn UML.
		for _, marker := range []string{`id="solid"`, `id="open"`} {
			if !strings.Contains(out, marker) {
				t.Errorf("%s: thiếu marker %s", s.Name, marker)
			}
		}
	}
}

func TestSequenceDrawioIsWellFormed(t *testing.T) {
	for _, s := range sequences() {
		out := renderSequenceDrawio(s)
		if err := xml.Unmarshal([]byte(out), new(any)); err != nil {
			t.Errorf("%s: .drawio không hợp lệ: %v", s.Name, err)
		}
	}
}

// TestActivationBarsAreBalanced: thanh activation phải có chiều cao dương và
// nằm trong khung sơ đồ.
func TestActivationBarsAreBalanced(t *testing.T) {
	for _, s := range sequences() {
		g := seqLayout(s)
		for _, bar := range activations(s, g) {
			if bar.y2 < bar.y1 {
				t.Errorf("%s: activation trên %q có y2 < y1 (%d < %d)",
					s.Name, bar.lifeline, bar.y2, bar.y1)
			}
			if bar.y1 < seqHeadY+seqHeadH {
				t.Errorf("%s: activation trên %q bắt đầu phía trên hộp tên", s.Name, bar.lifeline)
			}
			if bar.y2 > g.height {
				t.Errorf("%s: activation trên %q tràn khỏi canvas", s.Name, bar.lifeline)
			}
		}
	}
}

// TestSequenceCanvasIsReasonable: quá nhiều lifeline thì sơ đồ rộng không xem nổi.
func TestSequenceCanvasIsReasonable(t *testing.T) {
	const maxW, maxH = 2000, 2200
	for _, s := range sequences() {
		g := seqLayout(s)
		if g.width > maxW || g.height > maxH {
			t.Errorf("%s: canvas %dx%d vượt ngưỡng %dx%d — tách bớt lifeline hoặc message",
				s.Name, g.width, g.height, maxW, maxH)
		}
	}
}

// TestEveryFlowDocHasSequence: mỗi tài liệu trong docs/flows/ phải có đúng một
// sequence cùng tên gốc.
func TestEveryFlowDocHasSequence(t *testing.T) {
	have := map[string]bool{}
	for _, s := range sequences() {
		have[s.Name] = true
	}
	for _, want := range []string{
		"matching-notification-flow", "driver-onboarding-flow", "shipper-order-flow",
		"driver-location-flow", "authentication-flow", "error-handling-flow",
	} {
		if !have[want] {
			t.Errorf("docs/flows/%s.md thiếu sequence diagram", want)
		}
	}
}

// TestFlowsAreNotComponentDiagrams khoá lại quyết định thiết kế: flow phải là
// sequence (có trục thời gian), không được quay lại kiểu sơ đồ hộp.
func TestFlowsAreNotComponentDiagrams(t *testing.T) {
	flows := flowNames()
	for _, d := range diagrams() {
		if flows[d.Name] {
			t.Errorf("%q là flow nên phải khai báo trong sequences(), không phải diagrams()", d.Name)
		}
	}
}

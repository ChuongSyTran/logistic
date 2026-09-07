// Command doclint kiểm tra tài liệu trong docs/ có tuân thủ quy ước không.
//
// Vì sao cần: tài liệu không được biên dịch, nên đổi tên hay di chuyển file là
// link chết im lặng — không ai biết cho tới khi có người bấm vào. Repo này đã
// từng như vậy: tài liệu ghi `/api/user/v1/register` trong khi route thật đã đổi
// từ lâu, và các link tới `wallet_service/...` gãy sau khi file được chuyển
// xuống docs/reference/.
//
// Kiểm tra ba thứ:
//
//	1. Mọi liên kết tương đối trong .md đều trỏ tới file có thật.
//	2. Tên file theo kebab-case (trừ README.md).
//	3. Mỗi file .md mở đầu bằng một tiêu đề "# ".
//
// Chạy:  make docs-lint
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// docsRoot là thư mục được kiểm tra; README.md ở gốc repo cũng được đưa vào.
const docsRoot = "docs"

// skipDirs là các thư mục không phải tài liệu của repo.
var skipDirs = map[string]bool{
	"programmingBook": true, // thư viện sách cá nhân, đã gitignore
	"rendered":        true, // HTML xuất ra, không áp quy ước markdown
	"diagrams":        true, // file nguồn drawio
}

var (
	// [chữ hiển thị](đường/dẫn)  — bỏ qua link ngoài và anchor thuần
	linkRe = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	// kebab-case: chữ thường, số, gạch nối
	kebabRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*\.[a-z0-9]+$`)
)

type problem struct {
	file string
	line int
	msg  string
}

func main() {
	var problems []problem

	files, err := collectDocs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "doclint: %v\n", err)
		os.Exit(2)
	}

	for _, f := range files {
		problems = append(problems, checkNaming(f)...)
		p, err := checkContent(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "doclint: đọc %s: %v\n", f, err)
			os.Exit(2)
		}
		problems = append(problems, p...)
	}

	if len(problems) == 0 {
		fmt.Printf("doclint: OK — đã kiểm tra %d file tài liệu.\n", len(files))
		return
	}

	for _, p := range problems {
		if p.line > 0 {
			fmt.Printf("  %s:%d  %s\n", p.file, p.line, p.msg)
		} else {
			fmt.Printf("  %s  %s\n", p.file, p.msg)
		}
	}
	fmt.Printf("\ndoclint: phát hiện %d vấn đề.\n", len(problems))
	os.Exit(1)
}

func collectDocs() ([]string, error) {
	var out []string

	if _, err := os.Stat("README.md"); err == nil {
		out = append(out, "README.md")
	}

	err := filepath.WalkDir(docsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") {
			out = append(out, filepath.ToSlash(path))
		}
		return nil
	})
	return out, err
}

func checkNaming(path string) []problem {
	name := filepath.Base(path)
	if name == "README.md" {
		return nil // ngoại lệ có chủ ý: GitHub tự render file này
	}
	if !kebabRe.MatchString(name) {
		return []problem{{
			file: path,
			msg:  fmt.Sprintf("tên file %q không phải kebab-case (xem docs/README.md § Quy ước đặt tên)", name),
		}}
	}
	return nil
}

func checkContent(path string) ([]problem, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		problems  []problem
		dir       = filepath.Dir(path)
		inFence   bool
		lineNo    int
		sawHeader bool
		firstText bool
	)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // reference/*.md rất dài

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		// Bỏ qua nội dung trong code fence: ví dụ minh hoạ không phải link thật.
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		if !firstText && strings.TrimSpace(line) != "" {
			firstText = true
			sawHeader = strings.HasPrefix(line, "# ")
		}

		for _, m := range linkRe.FindAllStringSubmatch(line, -1) {
			target := strings.TrimSpace(m[1])
			if !isLocalPath(target) {
				continue
			}
			// Bỏ phần anchor "#muc-luc" trước khi kiểm tra file.
			if i := strings.IndexByte(target, '#'); i >= 0 {
				target = target[:i]
			}
			if target == "" {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, target)); err != nil {
				problems = append(problems, problem{
					file: path, line: lineNo,
					msg: fmt.Sprintf("liên kết chết -> %s", target),
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if firstText && !sawHeader {
		problems = append(problems, problem{
			file: path, line: 1,
			msg: `thiếu tiêu đề "# " ở dòng nội dung đầu tiên`,
		})
	}

	return problems, nil
}

// isLocalPath loại bỏ URL ngoài, mailto và anchor trong cùng trang.
func isLocalPath(target string) bool {
	switch {
	case target == "",
		strings.HasPrefix(target, "#"),
		strings.HasPrefix(target, "http://"),
		strings.HasPrefix(target, "https://"),
		strings.HasPrefix(target, "mailto:"),
		strings.HasPrefix(target, "//"):
		return false
	}
	return true
}

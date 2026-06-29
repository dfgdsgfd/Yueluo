package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizePostContentPreservesFixtureMarkdownAndDropsFixtureHTML(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "..", "testdata", "Full-Markdown.md")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read Markdown fixture: %v", err)
	}

	got := sanitizePostContent(string(raw))
	for _, needle := range []string{
		"# ✨ Markdown 演示文档（完整版）",
		"```mermaid",
		"sequenceDiagram",
		"$$",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("fixture Markdown feature %q was removed", needle)
		}
	}
	for _, needle := range []string{"```html\n<div class=\"card\">", "```html\n<video controls width=\"600\">"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("fixture fenced HTML code %q was not preserved", needle)
		}
	}
	for _, needle := range []string{"<details", "<summary", "</details>", "<u>"} {
		if strings.Contains(got, needle) {
			t.Fatalf("fixture raw HTML %q survived", needle)
		}
	}
}

func TestSanitizePostContentPreservesMarkdownWithoutHTML(t *testing.T) {
	want := "# Title\n\nBody with **bold**, [link](https://example.com), and `code`.\n\n```html\n<script>alert(1)</script>\n```"
	got := sanitizePostContent(want)
	if got != want {
		t.Fatalf("Markdown changed during backend normalization:\ngot  %q\nwant %q", got, want)
	}
}

func TestSanitizePostContentNormalizesLineEndingsAndNUL(t *testing.T) {
	got := sanitizePostContent("# Title\r\n\r\nBody\x00\r\n")
	if got != "# Title\n\nBody" {
		t.Fatalf("normalized Markdown = %q", got)
	}
}

func TestSanitizeCommentContentDropsPlainHTML(t *testing.T) {
	got := sanitizeCommentContent("正常评论\n<script>alert(1)</script><b onclick=\"x()\">加粗</b>")
	if got != "正常评论\n加粗" {
		t.Fatalf("sanitized comment = %q", got)
	}
}

func TestSanitizePlainSubmittedTextKeepsNormalComparisons(t *testing.T) {
	got := sanitizePlainSubmittedText("1 < 2 && 3 > 2")
	if got != "1 < 2 && 3 > 2" {
		t.Fatalf("plain comparison text = %q", got)
	}
}

func TestPostContentLengthCountsMarkdownRunesAndRichTextContent(t *testing.T) {
	if got := postContentLength("# 标题\n\n😀正文"); got != 9 {
		t.Fatalf("Markdown length = %d, want 9", got)
	}
	if got := postContentLength(`<h2>标题</h2><p>😀正文 <strong>加粗</strong></p>`); got != 17 {
		t.Fatalf("converted markdown length = %d, want 17", got)
	}
}

func TestSanitizePostContentDropsUnsafeMarkdownURLs(t *testing.T) {
	input := []string{
		"# Safe",
		"[good](https://example.com)",
		"[bad](javascript:alert(1))",
		"![bad image](data:text/html;base64,PHNjcmlwdD4=)",
		"<javascript:alert(1)>",
		"[ref]: javascript:alert(1)",
		"",
		"```markdown",
		"[code](javascript:alert(1))",
		"```",
	}
	got := sanitizePostContent(strings.Join(input, "\n"))
	if !strings.Contains(got, "[good](https://example.com)") {
		t.Fatalf("safe markdown URL was removed: %q", got)
	}
	for _, needle := range []string{"[bad](javascript:", "![bad image](data:text/html", "<javascript:", "[ref]:"} {
		if strings.Contains(got, needle) {
			t.Fatalf("unsafe markdown URL %q survived: %q", needle, got)
		}
	}
	if !strings.Contains(got, "[code](javascript:alert(1))") {
		t.Fatalf("fenced code was not preserved: %q", got)
	}
}

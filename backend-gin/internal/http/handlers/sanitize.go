package handlers

import (
	"strings"
	"unicode/utf8"

	"yuem-go/backend-gin/internal/contentformat"
)

func sanitizePostContent(content string) string {
	return contentformat.SanitizeMarkdown(content)
}

func sanitizeMarkdownSubmittedText(content string) string {
	return contentformat.SanitizeMarkdown(content)
}

func sanitizeCommentContent(content string) string {
	return contentformat.SanitizeMarkdown(content)
}

func sanitizePlainSubmittedText(content string) string {
	return contentformat.SanitizePlainText(content)
}

func normalizeSubmittedText(content string) string {
	return contentformat.NormalizeText(content)
}

func postContentLength(content string) int {
	content = strings.TrimSpace(contentformat.SanitizeMarkdown(content))
	return utf8.RuneCountInString(content)
}

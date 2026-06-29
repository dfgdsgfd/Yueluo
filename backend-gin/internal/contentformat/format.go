package contentformat

import (
	"bytes"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/microcosm-cc/bluemonday"
	nethtml "golang.org/x/net/html"
)

var (
	htmlTagRE           = regexp.MustCompile(`(?i)<\s*(?:/?[a-z][a-z0-9:-]*|!|\?)`)
	htmlFragmentStartRE = regexp.MustCompile(`(?is)^\s*<(?:!doctype|html|body|article|section|main|div|p|h[1-6]|blockquote|pre|ul|ol|li|table|thead|tbody|tr|td|th|span|strong|b|em|i|u|s|del|a|br|hr|code)\b`)
	markdownPolicy      *bluemonday.Policy
	markdownPolicyOnce  sync.Once
	plainPolicy         *bluemonday.Policy
	plainPolicyOnce     sync.Once
	mdConverter         *converter.Converter
	mdConverterOnce     sync.Once
)

func NormalizeText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\x00", "")
	return strings.TrimSpace(value)
}

func IsHTMLLike(value string) bool {
	return htmlTagRE.MatchString(value)
}

func SanitizeMarkdown(value string) string {
	value = NormalizeText(value)
	if value == "" {
		return ""
	}
	if !containsHTMLOutsideFencedCode(value) {
		return sanitizeMarkdownSource(value)
	}
	if looksLikeHTMLFragment(value) {
		markdown, err := HTMLToMarkdown(value)
		if err != nil {
			return SanitizePlainText(value)
		}
		return sanitizeMarkdownSource(markdown)
	}
	return sanitizeMarkdownSource(sanitizeMixedMarkdownHTML(value))
}

func containsHTMLOutsideFencedCode(value string) bool {
	inFence := false
	for line := range strings.SplitSeq(value, "\n") {
		if isFenceLine(line) {
			inFence = !inFence
			continue
		}
		if !inFence && htmlTagRE.MatchString(line) {
			return true
		}
	}
	return false
}

func looksLikeHTMLFragment(value string) bool {
	return htmlFragmentStartRE.MatchString(value)
}

func SanitizePlainText(value string) string {
	value = NormalizeText(value)
	if value == "" || !IsHTMLLike(value) {
		return value
	}
	return strings.TrimSpace(plainTextPolicy().Sanitize(value))
}

func HTMLToMarkdown(value string) (string, error) {
	value = NormalizeText(value)
	if value == "" {
		return "", nil
	}
	sanitized := markdownHTMLPolicy().Sanitize(value)
	if strings.TrimSpace(sanitized) == "" {
		return "", nil
	}
	prepared := prepareHTMLForMarkdown(sanitized)
	markdown, err := markdownConverter().ConvertString(prepared)
	if err != nil {
		return "", err
	}
	return normalizeMarkdownOutput(markdown), nil
}

func markdownConverter() *converter.Converter {
	mdConverterOnce.Do(func() {
		mdConverter = converter.NewConverter(
			converter.WithPlugins(
				base.NewBasePlugin(),
				commonmark.NewCommonmarkPlugin(
					commonmark.WithHeadingStyle("atx"),
					commonmark.WithHorizontalRule("---"),
					commonmark.WithBulletListMarker("-"),
					commonmark.WithListEndComment(false),
				),
				strikethrough.NewStrikethroughPlugin(),
				table.NewTablePlugin(),
			),
		)
	})
	return mdConverter
}

func markdownHTMLPolicy() *bluemonday.Policy {
	markdownPolicyOnce.Do(func() {
		p := bluemonday.NewPolicy()
		p.AllowElements(
			"a", "blockquote", "br", "code", "del", "div", "em", "h1", "h2", "h3", "h4", "h5", "h6",
			"hr", "li", "ol", "p", "pre", "s", "span", "strong", "table", "tbody", "td", "th", "thead", "tr", "u", "ul",
		)
		p.AllowAttrs("href").OnElements("a")
		p.AllowAttrs("colspan", "rowspan").Matching(regexp.MustCompile(`^[0-9]{1,3}$`)).OnElements("td", "th")
		p.AllowURLSchemes("http", "https", "mailto")
		p.AllowRelativeURLs(true)
		markdownPolicy = p
	})
	return markdownPolicy
}

func plainTextPolicy() *bluemonday.Policy {
	plainPolicyOnce.Do(func() {
		plainPolicy = bluemonday.StrictPolicy()
	})
	return plainPolicy
}

func prepareHTMLForMarkdown(value string) string {
	nodes, err := nethtml.ParseFragment(strings.NewReader(value), &nethtml.Node{Type: nethtml.ElementNode, Data: "div"})
	if err != nil {
		return value
	}
	wrapper := &nethtml.Node{Type: nethtml.ElementNode, Data: "div"}
	for _, node := range nodes {
		wrapper.AppendChild(node)
	}
	prepareMarkdownNode(wrapper)
	var out bytes.Buffer
	for child := wrapper.FirstChild; child != nil; child = child.NextSibling {
		_ = nethtml.Render(&out, child)
	}
	return out.String()
}

func prepareMarkdownNode(node *nethtml.Node) {
	if node.Type == nethtml.ElementNode {
		tag := strings.ToLower(node.Data)
		switch tag {
		case "a":
			node.Attr = sanitizeAnchorAttrs(node.Attr)
		case "u":
			node.Data = "span"
		case "span":
			if markdownSpanShouldUnwrap(node) {
				unwrapNode(node)
			}
		}
	}
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		prepareMarkdownNode(child)
		child = next
	}
}

func sanitizeAnchorAttrs(attrs []nethtml.Attribute) []nethtml.Attribute {
	for _, attr := range attrs {
		if strings.EqualFold(attr.Key, "href") && safeURL(attr.Val) {
			return []nethtml.Attribute{{Key: "href", Val: strings.TrimSpace(attr.Val)}}
		}
	}
	return nil
}

func markdownSpanShouldUnwrap(node *nethtml.Node) bool {
	if node == nil || node.Type != nethtml.ElementNode || !strings.EqualFold(node.Data, "span") {
		return false
	}
	return len(node.Attr) == 0
}

func unwrapNode(node *nethtml.Node) {
	parent := node.Parent
	if parent == nil {
		return
	}
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		node.RemoveChild(child)
		parent.InsertBefore(child, node)
		child = next
	}
	parent.RemoveChild(node)
}

func safeURL(raw string) bool {
	raw = strings.TrimSpace(nethtml.UnescapeString(raw))
	if raw == "" {
		return false
	}
	if strings.ContainsFunc(raw, func(r rune) bool {
		return unicode.IsControl(r) || unicode.IsSpace(r)
	}) {
		return false
	}
	if strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "/") {
		if strings.HasPrefix(raw, "//") {
			return false
		}
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Scheme == "" {
		return parsed.Host == ""
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "mailto":
		return true
	default:
		return false
	}
}

func sanitizeMarkdownSource(value string) string {
	lines := strings.Split(value, "\n")
	inFence := false
	for idx, line := range lines {
		if isFenceLine(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		lines[idx] = sanitizeMarkdownLine(line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func sanitizeMixedMarkdownHTML(value string) string {
	lines := strings.Split(value, "\n")
	inFence := false
	for idx, line := range lines {
		if isFenceLine(line) {
			inFence = !inFence
			continue
		}
		if inFence || !htmlTagRE.MatchString(line) {
			continue
		}
		converted, err := HTMLToMarkdown(line)
		if err != nil {
			converted = SanitizePlainText(line)
		}
		lines[idx] = converted
	}
	return strings.Join(lines, "\n")
}

func sanitizeMarkdownLine(line string) string {
	if sanitized, ok := sanitizeReferenceDefinition(line); ok {
		return sanitized
	}
	return sanitizeAutolinks(sanitizeInlineLinks(line))
}

func sanitizeReferenceDefinition(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if len(line)-len(trimmed) > 3 || !strings.HasPrefix(trimmed, "[") {
		return line, false
	}
	labelEnd := findClosingBracket(trimmed, 0)
	if labelEnd < 0 || labelEnd+1 >= len(trimmed) || trimmed[labelEnd+1] != ':' {
		return line, false
	}
	destination := strings.TrimSpace(trimmed[labelEnd+2:])
	if destination == "" || safeMarkdownDestination(destination) {
		return line, true
	}
	return "", true
}

func sanitizeInlineLinks(line string) string {
	var out strings.Builder
	for idx := 0; idx < len(line); {
		if line[idx] == '\\' && idx+1 < len(line) {
			out.WriteString(line[idx : idx+2])
			idx += 2
			continue
		}
		linkStart := idx
		labelStart := idx
		if line[idx] == '!' && idx+1 < len(line) && line[idx+1] == '[' {
			labelStart = idx + 1
		} else if line[idx] != '[' {
			out.WriteByte(line[idx])
			idx++
			continue
		}
		labelEnd := findClosingBracket(line, labelStart)
		if labelEnd < 0 || labelEnd+1 >= len(line) || line[labelEnd+1] != '(' {
			out.WriteByte(line[idx])
			idx++
			continue
		}
		destinationEnd := findClosingParen(line, labelEnd+2)
		if destinationEnd < 0 {
			out.WriteByte(line[idx])
			idx++
			continue
		}
		if safeMarkdownDestination(line[labelEnd+2 : destinationEnd]) {
			out.WriteString(line[linkStart : destinationEnd+1])
		} else {
			out.WriteString(line[labelStart+1 : labelEnd])
		}
		idx = destinationEnd + 1
	}
	return out.String()
}

func sanitizeAutolinks(line string) string {
	var out strings.Builder
	for idx := 0; idx < len(line); {
		if line[idx] != '<' {
			out.WriteByte(line[idx])
			idx++
			continue
		}
		end := strings.IndexByte(line[idx+1:], '>')
		if end < 0 {
			out.WriteByte(line[idx])
			idx++
			continue
		}
		end += idx + 1
		destination := line[idx+1 : end]
		if strings.Contains(destination, ":") && !safeURL(destination) {
			out.WriteString(destination)
		} else {
			out.WriteString(line[idx : end+1])
		}
		idx = end + 1
	}
	return out.String()
}

func safeMarkdownDestination(value string) bool {
	return safeURL(markdownDestinationURL(value))
}

func markdownDestinationURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "<") {
		if end := strings.IndexByte(value, '>'); end > 0 {
			return value[1:end]
		}
	}
	depth := 0
	for idx, r := range value {
		if r == '\\' {
			continue
		}
		if unicode.IsSpace(r) && depth == 0 {
			return value[:idx]
		}
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		}
	}
	return value
}

func findClosingBracket(value string, start int) int {
	depth := 0
	for idx := start + 1; idx < len(value); idx++ {
		if value[idx] == '\\' {
			idx++
			continue
		}
		switch value[idx] {
		case '[':
			depth++
		case ']':
			if depth == 0 {
				return idx
			}
			depth--
		}
	}
	return -1
}

func findClosingParen(value string, start int) int {
	depth := 0
	for idx := start; idx < len(value); idx++ {
		if value[idx] == '\\' {
			idx++
			continue
		}
		switch value[idx] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return idx
			}
			depth--
		}
	}
	return -1
}

func isFenceLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

func normalizeMarkdownOutput(value string) string {
	value = NormalizeText(value)
	value = strings.ReplaceAll(value, "\\[", "[")
	value = strings.ReplaceAll(value, "\\]", "]")
	value = regexp.MustCompile(`[ \t]+\n`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`\n{3,}`).ReplaceAllString(value, "\n\n")
	return strings.TrimSpace(value)
}

package services

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	aiFencePattern        = regexp.MustCompile("(?is)^\\s*```(?:markdown|md|text)?\\s*\\n([\\s\\S]*?)\\n```\\s*$")
	aiLeakLinePattern     = regexp.MustCompile("(?i)^(?:\\s{0,3}(?:#{1,6}\\s*)?)?(?:prompt|system prompt|user prompt|instructions?|analysis|reasoning|thinking|thought process|chain of thought|思考|思考过程|推理|推理过程|提示词|系统提示词|用户提示词|分析)\\s*[:：]\\s*")
	aiOpeningMetaPattern  = regexp.MustCompile("(?i)^(?:here(?:'s| is)\\b.*?:|sure[,，]?\\s*|当然[,，]?\\s*|以下(?:是|为).*?[:：]|下面(?:是|为).*?[:：])\\s*$")
	aiPromptMarkerPattern = regexp.MustCompile("(?i)\\b(?:locale|chunk|input|post context|requirements)\\s*:\\s*")
	aiThinkTagPattern     = regexp.MustCompile("(?is)<think>.*?</think>")
)

func sanitizeAITextOutput(value string, taskType string) string {
	text := strings.TrimSpace(sanitizeAIDBText(value))
	if text == "" {
		return ""
	}
	text = aiThinkTagPattern.ReplaceAllString(text, "")
	if match := aiFencePattern.FindStringSubmatch(text); len(match) == 2 {
		text = strings.TrimSpace(match[1])
	}
	text = stripAILeakBlocks(text)
	if isAITextTransformTask(taskType) {
		text = ensureMarkdownLongformIndex(text)
	}
	return strings.TrimSpace(text)
}

type aiStreamOutputFilter struct {
	inFence           bool
	inThink           bool
	pending           string
	skippingMetaBlock bool
	wroteContent      bool
}

func newAIStreamOutputFilter() *aiStreamOutputFilter {
	return &aiStreamOutputFilter{}
}

func (f *aiStreamOutputFilter) Process(delta string) string {
	if f == nil || delta == "" {
		return delta
	}
	f.pending += strings.ReplaceAll(delta, "\r\n", "\n")
	var out strings.Builder
	for {
		index := strings.IndexByte(f.pending, '\n')
		if index < 0 {
			break
		}
		line := f.pending[:index+1]
		f.pending = f.pending[index+1:]
		out.WriteString(f.filterLine(line))
	}
	if f.pending != "" && f.canEmitPendingPartial() {
		line := f.pending
		f.pending = ""
		out.WriteString(f.filterLine(line))
	}
	return out.String()
}

func (f *aiStreamOutputFilter) Flush() string {
	if f == nil || f.pending == "" {
		return ""
	}
	line := f.pending
	f.pending = ""
	return f.filterLine(line)
}

func (f *aiStreamOutputFilter) filterLine(line string) string {
	cleaned := f.stripThinkTags(line)
	trimmed := strings.TrimSpace(cleaned)
	if trimmed == "" {
		if f.skippingMetaBlock {
			f.skippingMetaBlock = false
			return ""
		}
		if f.inThink {
			return ""
		}
		if !f.wroteContent {
			return ""
		}
		return cleaned
	}
	if strings.HasPrefix(trimmed, "```") {
		f.inFence = !f.inFence
		f.wroteContent = true
		return cleaned
	}
	if f.inFence {
		f.wroteContent = true
		return cleaned
	}
	if isAILeakLine(trimmed) {
		f.skippingMetaBlock = true
		return ""
	}
	if f.skippingMetaBlock && looksLikePromptContinuation(trimmed) {
		return ""
	}
	f.skippingMetaBlock = false
	if !f.wroteContent && aiOpeningMetaPattern.MatchString(trimmed) {
		return ""
	}
	f.wroteContent = true
	return cleaned
}

func (f *aiStreamOutputFilter) canEmitPendingPartial() bool {
	if f.inThink {
		return false
	}
	trimmed := strings.TrimSpace(f.pending)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix("<think>", lower) || strings.HasPrefix("</think>", lower) {
		return false
	}
	if strings.Contains(lower, "<think") || strings.Contains(lower, "</think") {
		return strings.Contains(lower, "<think>") && strings.Contains(lower, "</think>")
	}
	if strings.HasPrefix("```", trimmed) {
		return false
	}
	if aiLeakLinePattern.MatchString(trimmed) || aiOpeningMetaPattern.MatchString(trimmed) {
		return false
	}
	if !f.inFence {
		if isAILeakLinePrefix(lower) || isAIOpeningMetaPrefix(lower) {
			return false
		}
	}
	return true
}

func (f *aiStreamOutputFilter) stripThinkTags(line string) string {
	var out strings.Builder
	rest := line
	for rest != "" {
		lower := strings.ToLower(rest)
		if f.inThink {
			end := strings.Index(lower, "</think>")
			if end < 0 {
				return out.String()
			}
			rest = rest[end+len("</think>"):]
			f.inThink = false
			continue
		}
		start := strings.Index(lower, "<think>")
		if start < 0 {
			out.WriteString(rest)
			break
		}
		out.WriteString(rest[:start])
		rest = rest[start+len("<think>"):]
		f.inThink = true
	}
	return out.String()
}

func isAILeakLinePrefix(lowerLine string) bool {
	labels := []string{
		"prompt", "system prompt", "user prompt", "instruction", "instructions",
		"analysis", "reasoning", "thinking", "thought process", "chain of thought",
		"思考", "思考过程", "推理", "推理过程", "提示词", "系统提示词", "用户提示词", "分析",
	}
	for _, label := range labels {
		label = strings.ToLower(label)
		if strings.HasPrefix(label+":", lowerLine) || strings.HasPrefix(label+"：", lowerLine) || strings.HasPrefix(label, lowerLine) {
			return true
		}
	}
	return false
}

func isAIOpeningMetaPrefix(lowerLine string) bool {
	prefixes := []string{"here", "here's", "here is", "sure", "当然", "以下", "下面"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(prefix, lowerLine) || strings.HasPrefix(lowerLine, prefix) {
			return true
		}
	}
	return false
}

func isAITextTransformTask(taskType string) bool {
	switch strings.TrimSpace(taskType) {
	case AITaskFormatMarkdown, AITaskPostPolish, AITaskPostCustomGenerate:
		return true
	default:
		return false
	}
}

func shouldEchoBlankAIChunk(requestTaskType string, templateTaskType string) bool {
	return isAITextTransformTask(requestTaskType) && isAITextTransformTask(templateTaskType)
}

func stripAILeakBlocks(value string) string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	skippingMetaBlock := false
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			out = append(out, line)
			continue
		}
		if inFence {
			out = append(out, line)
			continue
		}
		if trimmed == "" {
			if !skippingMetaBlock {
				out = append(out, line)
			}
			skippingMetaBlock = false
			continue
		}
		if isAILeakLine(trimmed) {
			skippingMetaBlock = true
			continue
		}
		if skippingMetaBlock && looksLikePromptContinuation(trimmed) {
			continue
		}
		skippingMetaBlock = false
		if len(out) == 0 && aiOpeningMetaPattern.MatchString(trimmed) {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(compactExcessBlankLines(strings.Join(out, "\n")))
}

func isAILeakLine(line string) bool {
	if aiLeakLinePattern.MatchString(line) {
		return true
	}
	lower := strings.ToLower(line)
	return strings.HasPrefix(lower, "<think>") ||
		strings.HasPrefix(lower, "</think>") ||
		strings.Contains(lower, "chain-of-thought")
}

func looksLikePromptContinuation(line string) bool {
	if strings.HasPrefix(line, "{{") || strings.HasPrefix(line, "}}") {
		return true
	}
	return aiPromptMarkerPattern.MatchString(line)
}

func compactExcessBlankLines(value string) string {
	lines := strings.Split(value, "\n")
	out := make([]string, 0, len(lines))
	blankCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount > 2 {
				continue
			}
		} else {
			blankCount = 0
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func ensureMarkdownLongformIndex(value string) string {
	text := strings.TrimSpace(value)
	if text == "" || strings.Contains(text, "<!-- ai-longform-index -->") || strings.Contains(text, "](#") {
		return text
	}
	lines := strings.Split(text, "\n")
	headings := make([]markdownHeadingCandidate, 0)
	for _, line := range lines {
		level, title := parseMarkdownHeading(line)
		if level == 2 && title != "" {
			headings = append(headings, markdownHeadingCandidate{Title: title})
		}
	}
	if len(headings) < 4 && utf8.RuneCountInString(text) >= 8000 {
		headings = inferLongformMarkdownSections(lines)
	}
	if len(headings) < 4 {
		return text
	}
	var builder strings.Builder
	builder.WriteString("<!-- ai-longform-index -->\n\n## ")
	builder.WriteString(aiIndexHeading(text))
	builder.WriteString("\n\n")
	seen := map[string]int{}
	for _, heading := range headings {
		slug := aiMarkdownSlug(heading.Title)
		if slug == "" {
			continue
		}
		seen[slug]++
		if seen[slug] > 1 {
			slug = slug + "-" + intToString(seen[slug])
		}
		builder.WriteString("- [")
		builder.WriteString(heading.Title)
		builder.WriteString("](#")
		builder.WriteString(slug)
		builder.WriteString(")\n")
	}
	builder.WriteString("\n")
	builder.WriteString(text)
	return builder.String()
}

func aiIndexHeading(value string) string {
	cjkCount := 0
	latinCount := 0
	for _, r := range value {
		switch {
		case r >= 0x4e00 && r <= 0x9fff:
			cjkCount++
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			latinCount++
		}
		if cjkCount+latinCount > 300 {
			break
		}
	}
	if cjkCount > latinCount {
		return "目录"
	}
	return "Index"
}

type markdownHeadingCandidate struct {
	Title string
}

func parseMarkdownHeading(line string) (int, string) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return 0, ""
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || level >= len(trimmed) || trimmed[level] != ' ' {
		return 0, ""
	}
	return level, strings.TrimSpace(trimmed[level+1:])
}

func inferLongformMarkdownSections(lines []string) []markdownHeadingCandidate {
	out := make([]markdownHeadingCandidate, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || len([]rune(trimmed)) > 80 {
			continue
		}
		if looksLikeLongformSectionTitle(trimmed) {
			out = append(out, markdownHeadingCandidate{Title: strings.Trim(trimmed, "#* 　")})
		}
	}
	return out
}

func looksLikeLongformSectionTitle(value string) bool {
	prefixes := []string{
		"第", "Chapter ", "CHAPTER ", "Episode ", "EPISODE ", "Part ", "PART ",
		"卷", "幕", "篇", "回",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func aiMarkdownSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r), unicode.IsMark(r):
			builder.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_' || r == '\t':
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func intToString(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	buf := [20]byte{}
	index := len(buf)
	for value > 0 {
		index--
		buf[index] = digits[value%10]
		value /= 10
	}
	return string(buf[index:])
}

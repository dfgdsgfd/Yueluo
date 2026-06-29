package services

import (
	"fmt"
	"maps"
	"strings"
	"unicode/utf8"
)

func SplitAITextChunks(input string, maxChars int) []string {
	input = strings.TrimSpace(strings.ReplaceAll(input, "\r\n", "\n"))
	if input == "" {
		return nil
	}
	if maxChars == 0 {
		return []string{input}
	}
	if maxChars < 500 {
		maxChars = 3000
	}
	paragraphs := splitParagraphs(input)
	chunks := make([]string, 0, len(paragraphs))
	var current strings.Builder
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		if utf8.RuneCountInString(paragraph) > maxChars {
			if strings.TrimSpace(current.String()) != "" {
				chunks = append(chunks, strings.TrimSpace(current.String()))
				current.Reset()
			}
			chunks = append(chunks, splitLongRunes(paragraph, maxChars)...)
			continue
		}
		nextLen := utf8.RuneCountInString(current.String()) + utf8.RuneCountInString(paragraph) + 2
		if current.Len() > 0 && nextLen > maxChars {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(paragraph)
	}
	if strings.TrimSpace(current.String()) != "" {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}
	return chunks
}

func splitParagraphs(input string) []string {
	lines := strings.Split(input, "\n")
	out := []string{}
	var current strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if strings.TrimSpace(current.String()) != "" {
				out = append(out, current.String())
				current.Reset()
			}
			continue
		}
		if current.Len() > 0 {
			current.WriteByte('\n')
		}
		current.WriteString(line)
	}
	if strings.TrimSpace(current.String()) != "" {
		out = append(out, current.String())
	}
	return out
}

func splitLongRunes(input string, maxChars int) []string {
	runes := []rune(input)
	out := []string{}
	for len(runes) > 0 {
		n := min(len(runes), maxChars)
		out = append(out, strings.TrimSpace(string(runes[:n])))
		runes = runes[n:]
	}
	return out
}

func MaskSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	runes := []rune(secret)
	if len(runes) <= 8 {
		return "••••"
	}
	return string(runes[:4]) + "••••" + string(runes[len(runes)-4:])
}

func renderAIPrompt(template string, input string, locale string, variables map[string]any) string {
	if strings.TrimSpace(template) == "" {
		template = "{{input}}"
	}
	return renderAIPromptTemplate(template, input, locale, variables)
}

func renderAIPromptTemplate(template string, input string, locale string, variables map[string]any) string {
	values := cloneVariables(variables)
	values["input"] = input
	values["locale"] = locale
	out := template
	for key, value := range values {
		out = strings.ReplaceAll(out, "{{"+key+"}}", fmt.Sprint(value))
	}
	return out
}

func cloneVariables(input map[string]any) map[string]any {
	out := map[string]any{}
	maps.Copy(out, input)
	return out
}

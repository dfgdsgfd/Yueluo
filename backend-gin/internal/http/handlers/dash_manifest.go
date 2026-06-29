package handlers

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/url"
	pathpkg "path"
	"strconv"
	"strings"
	"time"
)

var dashURLAttributeNames = map[string]bool{
	"href":           true,
	"index":          true,
	"initialization": true,
	"media":          true,
	"sourceURL":      true,
}

func (h NativeHandlers) signDASHManifestReferences(data []byte, manifestPath string, now time.Time) ([]byte, bool) {
	expiry := now.Add(h.fileSigningTTL()).Unix()
	signature := h.fileSignature(manifestPath, expiry)
	if signature == "" {
		return data, false
	}
	query := url.Values{}
	query.Set(fileSignatureExpiryParam, strconv.FormatInt(expiry, 10))
	query.Set(fileSignatureParam, signature)
	queryText := query.Encode()

	decoder := xml.NewDecoder(bytes.NewReader(data))
	var out bytes.Buffer
	encoder := xml.NewEncoder(&out)
	changed := false
	baseURLDepth := 0

	for {
		token, err := decoder.Token()
		if err != nil {
			if err != io.EOF {
				return data, false
			}
			break
		}

		switch typed := token.(type) {
		case xml.StartElement:
			if typed.Name.Local == "BaseURL" {
				baseURLDepth++
			}
			for i := range typed.Attr {
				if !dashURLAttributeNames[typed.Attr[i].Name.Local] {
					continue
				}
				rewritten, ok := h.signDASHReferenceURL(typed.Attr[i].Value, manifestPath, queryText)
				if ok {
					typed.Attr[i].Value = rewritten
					changed = true
				}
			}
			_ = encoder.EncodeToken(typed)
		case xml.EndElement:
			if typed.Name.Local == "BaseURL" && baseURLDepth > 0 {
				baseURLDepth--
			}
			_ = encoder.EncodeToken(typed)
		case xml.CharData:
			if baseURLDepth > 0 {
				rewritten, ok := h.signDASHReferenceURL(string(typed), manifestPath, queryText)
				if ok {
					token = xml.CharData([]byte(rewritten))
					changed = true
				}
			}
			_ = encoder.EncodeToken(token)
		default:
			_ = encoder.EncodeToken(token)
		}
	}
	if err := encoder.Flush(); err != nil || !changed {
		return data, changed
	}
	return out.Bytes(), true
}

func (h NativeHandlers) signDASHReferenceURL(raw string, manifestPath string, queryText string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "data:") || strings.HasPrefix(trimmed, "blob:") || strings.HasPrefix(trimmed, "urn:") {
		return raw, false
	}
	if strings.Contains(trimmed, "://") && !strings.Contains(trimmed, "/api/file/videos/") {
		return raw, false
	}

	pathPart, existingQuery, fragment := splitDASHURLParts(trimmed)
	if pathPart == "" || hasFileSignatureQuery(existingQuery) {
		return raw, false
	}
	canonicalPath := pathPart
	if parsed, err := url.Parse(pathPart); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		canonicalPath = parsed.Path
	}
	if !strings.HasPrefix(canonicalPath, "/") {
		canonicalPath = pathpkg.Join(pathpkg.Dir(manifestPath), canonicalPath)
	}
	if !strings.HasPrefix(canonicalPath, "/api/file/videos/") || !strings.EqualFold(pathpkg.Ext(canonicalPath), ".m4s") {
		return raw, false
	}

	return appendQueryToDASHURL(raw, queryText, fragment), true
}

func splitDASHURLParts(raw string) (string, string, string) {
	withoutFragment, fragment, _ := strings.Cut(raw, "#")
	pathPart, query, _ := strings.Cut(withoutFragment, "?")
	return pathPart, query, fragment
}

func hasFileSignatureQuery(query string) bool {
	if query == "" {
		return false
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return strings.Contains(query, fileSignatureExpiryParam+"=") || strings.Contains(query, fileSignatureParam+"=")
	}
	return values.Get(fileSignatureExpiryParam) != "" || values.Get(fileSignatureParam) != ""
}

func appendQueryToDASHURL(raw string, queryText string, fragment string) string {
	withoutFragment := raw
	if fragment != "" {
		withoutFragment, _, _ = strings.Cut(raw, "#")
	}
	separator := "?"
	if strings.Contains(withoutFragment, "?") {
		separator = "&"
	}
	rewritten := withoutFragment + separator + queryText
	if fragment != "" {
		rewritten += "#" + fragment
	}
	return rewritten
}

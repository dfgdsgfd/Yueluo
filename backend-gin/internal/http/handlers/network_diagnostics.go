package handlers

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
)

type networkDiagnosticsPayload struct {
	CDN        networkDiagnosticsCDN        `json:"cdn"`
	ClientIP   string                       `json:"clientIp"`
	Connection networkDiagnosticsConnection `json:"connection"`
	Forwarded  networkDiagnosticsForwarded  `json:"forwarded"`
	Headers    networkDiagnosticsHeaderSets `json:"headers"`
	Request    networkDiagnosticsRequest    `json:"request"`
	Server     networkDiagnosticsServer     `json:"server"`
	Timestamp  string                       `json:"timestamp"`
}

type networkDiagnosticsConnection struct {
	Local  networkDiagnosticsAddress `json:"local"`
	Remote networkDiagnosticsAddress `json:"remote"`
}

type networkDiagnosticsAddress struct {
	Address string `json:"address"`
	IP      string `json:"ip"`
	Port    string `json:"port"`
}

type networkDiagnosticsRequest struct {
	Host          string `json:"host"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	Protocol      string `json:"protocol"`
	ProtocolMajor int    `json:"protocolMajor"`
	ProtocolMinor int    `json:"protocolMinor"`
	Query         string `json:"query"`
	RequestURI    string `json:"requestUri"`
	Scheme        string `json:"scheme"`
	TLS           bool   `json:"tls"`
	TLSCipher     string `json:"tlsCipher"`
	TLSVersion    string `json:"tlsVersion"`
	UserAgent     string `json:"userAgent"`
}

type networkDiagnosticsServer struct {
	Hostname string `json:"hostname"`
}

type networkDiagnosticsForwarded struct {
	Chain    []string `json:"chain"`
	Host     string   `json:"host"`
	Method   string   `json:"method"`
	Port     string   `json:"port"`
	Protocol string   `json:"protocol"`
	URI      string   `json:"uri"`
}

type networkDiagnosticsHeaderSets struct {
	CDN        []networkDiagnosticsHeader `json:"cdn"`
	Diagnostic []networkDiagnosticsHeader `json:"diagnostic"`
	Forwarded  []networkDiagnosticsHeader `json:"forwarded"`
}

type networkDiagnosticsCDN struct {
	CacheStatus string                     `json:"cacheStatus"`
	CountryCode string                     `json:"countryCode"`
	Detected    bool                       `json:"detected"`
	Evidence    []networkDiagnosticsHeader `json:"evidence"`
	Provider    *string                    `json:"provider"`
	RayID       string                     `json:"rayId"`
}

type networkDiagnosticsHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

var networkForwardedHeaderNames = []string{
	"forwarded",
	"x-forwarded-for",
	"x-real-ip",
	"x-forwarded-proto",
	"x-forwarded-host",
	"x-forwarded-port",
	"x-forwarded-server",
	"x-forwarded-method",
	"x-forwarded-uri",
	"x-original-forwarded-for",
	"x-original-host",
	"x-original-uri",
}

var networkCDNHeaderNames = []string{
	"cdn-loop",
	"cf-cache-status",
	"cf-connecting-ip",
	"cf-connecting-ipv6",
	"cf-ipcountry",
	"cf-ray",
	"cf-visitor",
	"cloudfront-forwarded-proto",
	"cloudfront-viewer-country",
	"fastly-client-ip",
	"true-client-ip",
	"via",
	"x-amz-cf-id",
	"x-cache",
	"x-cdn",
	"x-nf-request-id",
	"x-nws-log-uuid",
	"x-served-by",
	"x-tencent-ua",
	"x-vercel-cache",
	"x-vercel-id",
	"x-vercel-ip-country",
}

var networkDiagnosticHeaderNames = []string{
	"accept",
	"accept-language",
	"host",
	"user-agent",
	"x-request-id",
	"x-correlation-id",
	"x-forwarded-for",
	"x-forwarded-host",
	"x-forwarded-proto",
	"x-forwarded-port",
	"x-forwarded-method",
	"x-forwarded-uri",
	"x-real-ip",
	"forwarded",
	"cf-ray",
	"cf-connecting-ip",
	"cf-ipcountry",
	"cdn-loop",
}

func (h NativeHandlers) NetworkDiagnostics(c *gin.Context) {
	request := c.Request
	local := splitNetworkAddress(requestLocalAddress(request))
	remote := splitNetworkAddress(request.RemoteAddr)
	hostname, _ := os.Hostname()
	cdn := detectNetworkDiagnosticsCDN(request.Header)
	forwardedChain := networkForwardedChain(request.Header)

	payload := networkDiagnosticsPayload{
		CDN:      cdn,
		ClientIP: firstNonEmptyNetworkValue(networkObservedClientIP(request.Header), c.ClientIP(), remote.IP),
		Connection: networkDiagnosticsConnection{
			Local:  local,
			Remote: remote,
		},
		Forwarded: networkDiagnosticsForwarded{
			Chain:    forwardedChain,
			Host:     firstHeaderValue(request.Header, "x-forwarded-host", "x-original-host"),
			Method:   firstHeaderValue(request.Header, "x-forwarded-method"),
			Port:     firstHeaderValue(request.Header, "x-forwarded-port"),
			Protocol: firstHeaderValue(request.Header, "x-forwarded-proto"),
			URI:      firstHeaderValue(request.Header, "x-forwarded-uri", "x-original-uri"),
		},
		Headers: networkDiagnosticsHeaderSets{
			CDN:        pickNetworkHeaders(request.Header, networkCDNHeaderNames),
			Diagnostic: pickNetworkHeadersWithHost(request, networkDiagnosticHeaderNames),
			Forwarded:  pickNetworkHeaders(request.Header, networkForwardedHeaderNames),
		},
		Request: networkDiagnosticsRequest{
			Host:          request.Host,
			Method:        request.Method,
			Path:          request.URL.Path,
			Protocol:      request.Proto,
			ProtocolMajor: request.ProtoMajor,
			ProtocolMinor: request.ProtoMinor,
			Query:         request.URL.RawQuery,
			RequestURI:    request.RequestURI,
			Scheme:        networkRequestScheme(request),
			TLS:           request.TLS != nil,
			TLSCipher:     networkTLSCipher(request),
			TLSVersion:    networkTLSVersion(request),
			UserAgent:     request.UserAgent(),
		},
		Server: networkDiagnosticsServer{
			Hostname: hostname,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}

	response.Success(c, payload, "ok")
}

func requestLocalAddress(request *http.Request) string {
	addr, ok := request.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok || addr == nil {
		return ""
	}
	return addr.String()
}

func splitNetworkAddress(value string) networkDiagnosticsAddress {
	value = strings.TrimSpace(value)
	if value == "" {
		return networkDiagnosticsAddress{}
	}
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return networkDiagnosticsAddress{Address: value, IP: strings.Trim(value, "[]")}
	}
	return networkDiagnosticsAddress{
		Address: value,
		IP:      strings.Trim(host, "[]"),
		Port:    port,
	}
}

func networkRequestScheme(request *http.Request) string {
	if proto := strings.TrimSuffix(strings.ToLower(firstHeaderValue(request.Header, "x-forwarded-proto")), ":"); proto != "" {
		return proto
	}
	if request.TLS != nil {
		return "https"
	}
	if request.URL.Scheme != "" {
		return request.URL.Scheme
	}
	return "http"
}

func networkTLSVersion(request *http.Request) string {
	if request.TLS == nil {
		return ""
	}
	switch request.TLS.Version {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	default:
		return ""
	}
}

func networkTLSCipher(request *http.Request) string {
	if request.TLS == nil {
		return ""
	}
	return tls.CipherSuiteName(request.TLS.CipherSuite)
}

func detectNetworkDiagnosticsCDN(headers http.Header) networkDiagnosticsCDN {
	provider := detectNetworkDiagnosticsProvider(headers)
	evidence := pickNetworkHeaders(headers, networkCDNHeaderNames)
	return networkDiagnosticsCDN{
		CacheStatus: firstHeaderValue(headers, "cf-cache-status", "x-cache", "x-vercel-cache"),
		CountryCode: firstHeaderValue(headers,
			"cf-ipcountry",
			"cloudfront-viewer-country",
			"x-vercel-ip-country",
		),
		Detected: provider != "",
		Evidence: evidence,
		Provider: nullableNetworkString(provider),
		RayID: firstHeaderValue(headers,
			"cf-ray",
			"x-amz-cf-id",
			"x-nf-request-id",
			"x-vercel-id",
			"x-nws-log-uuid",
			"x-request-id",
		),
	}
}

func nullableNetworkString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func detectNetworkDiagnosticsProvider(headers http.Header) string {
	via := strings.ToLower(headers.Get("via"))
	xCache := strings.ToLower(headers.Get("x-cache"))
	xCDN := strings.ToLower(headers.Get("x-cdn"))
	cdnLoop := strings.ToLower(headers.Get("cdn-loop"))

	switch {
	case hasAnyNetworkHeader(headers, "cf-ray", "cf-cache-status", "cf-connecting-ip") || strings.Contains(cdnLoop, "cloudflare"):
		return "cloudflare"
	case hasAnyNetworkHeader(headers, "x-amz-cf-id", "cloudfront-viewer-country") || strings.Contains(via, "cloudfront"):
		return "cloudfront"
	case hasAnyNetworkHeader(headers, "fastly-client-ip", "x-served-by") || strings.Contains(xCache, "fastly"):
		return "fastly"
	case hasAnyNetworkHeader(headers, "x-vercel-id", "x-vercel-cache", "x-vercel-ip-country"):
		return "vercel"
	case hasAnyNetworkHeader(headers, "x-nf-request-id"):
		return "netlify"
	case hasAnyNetworkHeader(headers, "x-nws-log-uuid", "x-tencent-ua") || strings.Contains(xCDN, "tencent"):
		return "tencent"
	case strings.Contains(via, "kunlun") || strings.Contains(via, "alicdn") || strings.Contains(xCDN, "alibaba"):
		return "alibaba"
	case len(pickNetworkHeaders(headers, networkCDNHeaderNames)) > 0:
		return "unknown"
	default:
		return ""
	}
}

func networkObservedClientIP(headers http.Header) string {
	if value := firstHeaderValue(headers, "cf-connecting-ip", "true-client-ip", "x-real-ip", "x-client-ip", "fastly-client-ip", "x-vercel-forwarded-for"); value != "" {
		return value
	}
	if value := firstHeaderValue(headers, "x-forwarded-for"); value != "" {
		return splitNetworkList(value)[0]
	}
	return ""
}

func networkForwardedChain(headers http.Header) []string {
	values := append([]string{}, splitNetworkList(headers.Get("x-forwarded-for"))...)
	values = append(values, extractForwardedForValues(headers.Get("forwarded"))...)
	return uniqueNetworkValues(values)
}

func extractForwardedForValues(value string) []string {
	items := make([]string, 0)
	for _, entry := range splitNetworkList(value) {
		for part := range strings.SplitSeq(entry, ";") {
			key, raw, ok := strings.Cut(strings.TrimSpace(part), "=")
			if !ok || !strings.EqualFold(key, "for") {
				continue
			}
			items = append(items, strings.Trim(raw, `"`))
		}
	}
	return items
}

func splitNetworkList(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func uniqueNetworkValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func pickNetworkHeadersWithHost(request *http.Request, names []string) []networkDiagnosticsHeader {
	headers := pickNetworkHeaders(request.Header, names)
	for _, name := range names {
		if strings.EqualFold(name, "host") && request.Host != "" && request.Header.Get("host") == "" {
			headers = append(headers, networkDiagnosticsHeader{Name: name, Value: request.Host})
			break
		}
	}
	return headers
}

func pickNetworkHeaders(headers http.Header, names []string) []networkDiagnosticsHeader {
	picked := make([]networkDiagnosticsHeader, 0, len(names))
	for _, name := range names {
		if value := headers.Get(name); value != "" {
			picked = append(picked, networkDiagnosticsHeader{Name: name, Value: value})
		}
	}
	return picked
}

func firstHeaderValue(headers http.Header, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(headers.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

func hasAnyNetworkHeader(headers http.Header, names ...string) bool {
	for _, name := range names {
		if headers.Get(name) != "" {
			return true
		}
	}
	return false
}

func firstNonEmptyNetworkValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

package services

import (
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang"

	"yuem-go/backend-gin/internal/config"
)

type GeoIPCountry struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Flag string `json:"flag"`
}

type GeoIPService struct {
	cfg     config.GeoIPConfig
	mu      sync.Mutex
	reader  *maxminddb.Reader
	path    string
	modTime time.Time
	size    int64
}

type maxMindCountryRecord struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	RegisteredCountry struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"registered_country"`
	RepresentedCountry struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"represented_country"`
}

func NewGeoIPService(cfg config.GeoIPConfig) *GeoIPService {
	return &GeoIPService{cfg: cfg}
}

func (s *GeoIPService) CountryForIP(raw string) GeoIPCountry {
	addr, ok := parseIPAddr(raw)
	if !ok {
		return GeoIPCountry{}
	}
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
		return GeoIPCountry{Code: "LAN", Name: "Local network"}
	}
	reader, err := s.readerForCurrentFile()
	if err != nil || reader == nil {
		return GeoIPCountry{}
	}
	var record maxMindCountryRecord
	if err := reader.Lookup(net.IP(addr.AsSlice()), &record); err != nil {
		return GeoIPCountry{}
	}
	code, name := countryRecordValue(record.Country.ISOCode, record.Country.Names)
	if code == "" {
		code, name = countryRecordValue(record.RegisteredCountry.ISOCode, record.RegisteredCountry.Names)
	}
	if code == "" {
		code, name = countryRecordValue(record.RepresentedCountry.ISOCode, record.RepresentedCountry.Names)
	}
	if code == "" {
		return GeoIPCountry{}
	}
	return GeoIPCountry{Code: code, Name: name, Flag: countryFlag(code)}
}

func (s *GeoIPService) DatabasePath() string {
	if s == nil {
		return ""
	}
	return resolveSystemUpdatePath(defaultIfBlank(s.cfg.CountryMMDBPath, "data/geoip/Country.mmdb"))
}

func (s *GeoIPService) readerForCurrentFile() (*maxminddb.Reader, error) {
	if s == nil {
		return nil, os.ErrInvalid
	}
	path := s.DatabasePath()
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	path, _ = filepath.Abs(path)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reader != nil && s.path == path && s.modTime.Equal(info.ModTime()) && s.size == info.Size() {
		return s.reader, nil
	}
	if s.reader != nil {
		_ = s.reader.Close()
		s.reader = nil
	}
	reader, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	s.reader = reader
	s.path = path
	s.modTime = info.ModTime()
	s.size = info.Size()
	return reader, nil
}

func parseIPAddr(raw string) (netip.Addr, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return netip.Addr{}, false
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.Trim(value, "[]")
	addr, err := netip.ParseAddr(value)
	return addr, err == nil
}

func countryRecordValue(code string, names map[string]string) (string, string) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return "", ""
	}
	for _, key := range []string{"zh-CN", "zh", "en"} {
		if value := strings.TrimSpace(names[key]); value != "" {
			return code, value
		}
	}
	for _, value := range names {
		if strings.TrimSpace(value) != "" {
			return code, strings.TrimSpace(value)
		}
	}
	return code, code
}

func countryFlag(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 2 {
		return ""
	}
	first := rune(code[0])
	second := rune(code[1])
	if first < 'A' || first > 'Z' || second < 'A' || second > 'Z' {
		return ""
	}
	return string([]rune{0x1F1E6 + first - 'A', 0x1F1E6 + second - 'A'})
}

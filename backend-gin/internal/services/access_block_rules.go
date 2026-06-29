package services

import (
	"fmt"
	"net/netip"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"yuem-go/backend-gin/internal/domain"
)

func emptyAccessBlockSnapshot() *accessBlockSnapshot {
	return &accessBlockSnapshot{
		rules:      []AccessBlockRuleDTO{},
		ipExact:    map[netip.Addr][]compiledAccessBlockRule{},
		ipPrefixes: map[int][]compiledAccessBlockRule{},
	}
}

func buildAccessBlockSnapshot(rows []domain.AccessBlockRule) *accessBlockSnapshot {
	snapshot := emptyAccessBlockSnapshot()
	for _, row := range rows {
		dto := accessBlockRuleDTO(row)
		snapshot.rules = append(snapshot.rules, dto)
		if !row.Enabled {
			continue
		}
		compiled, ok := compileAccessBlockRule(dto)
		if !ok {
			continue
		}
		switch dto.MatchType {
		case AccessBlockMatchIP:
			snapshot.ipExact[compiled.addr] = append(snapshot.ipExact[compiled.addr], compiled)
		case AccessBlockMatchCIDR:
			snapshot.ipPrefixes[compiled.prefix.Bits()] = append(snapshot.ipPrefixes[compiled.prefix.Bits()], compiled)
		case AccessBlockMatchUAContains:
			snapshot.uaContains = append(snapshot.uaContains, compiled)
		case AccessBlockMatchUARegex:
			snapshot.uaRegex = append(snapshot.uaRegex, compiled)
		}
	}
	sortAccessBlockRules(snapshot.uaContains)
	sortAccessBlockRules(snapshot.uaRegex)
	for key, rules := range snapshot.ipExact {
		sortAccessBlockRules(rules)
		snapshot.ipExact[key] = rules
	}
	for key, rules := range snapshot.ipPrefixes {
		sortAccessBlockRules(rules)
		snapshot.ipPrefixes[key] = rules
	}
	return snapshot
}

func compileAccessBlockRule(dto AccessBlockRuleDTO) (compiledAccessBlockRule, bool) {
	rule := compiledAccessBlockRule{dto: dto, sortIndex: int(dto.ID)}
	switch dto.MatchType {
	case AccessBlockMatchIP:
		ip, ok := parseAccessBlockAddr(dto.Pattern)
		if !ok {
			return rule, false
		}
		rule.addr = ip
		rule.specific = ip.BitLen()
	case AccessBlockMatchCIDR:
		prefix, ok := parseAccessBlockPrefix(dto.Pattern)
		if !ok {
			return rule, false
		}
		rule.prefix = prefix
		rule.specific = prefix.Bits()
	case AccessBlockMatchUAContains:
		needle := strings.ToLower(strings.TrimSpace(dto.Pattern))
		if needle == "" {
			return rule, false
		}
		rule.contains = needle
	case AccessBlockMatchUARegex:
		re, err := regexp.Compile(dto.Pattern)
		if err != nil {
			return rule, false
		}
		rule.regex = re
	default:
		return rule, false
	}
	return rule, true
}

func sortAccessBlockRules(rules []compiledAccessBlockRule) {
	sort.SliceStable(rules, func(i, j int) bool {
		return accessBlockRuleLess(rules[i], rules[j])
	})
}

func accessBlockRuleLess(a compiledAccessBlockRule, b compiledAccessBlockRule) bool {
	if a.dto.Priority != b.dto.Priority {
		return a.dto.Priority < b.dto.Priority
	}
	if a.specific != b.specific {
		return a.specific > b.specific
	}
	if a.dto.ID != b.dto.ID {
		return a.dto.ID < b.dto.ID
	}
	return a.sortIndex < b.sortIndex
}

func matchedAccessBlockValue(rule compiledAccessBlockRule, input AccessBlockMatchInput) string {
	if rule.dto.Kind == AccessBlockKindIP {
		return strings.TrimSpace(input.IP)
	}
	return strings.TrimSpace(input.UserAgent)
}

func normalizeAccessBlockRuleInput(input AccessBlockRuleInput, existing *domain.AccessBlockRule) (domain.AccessBlockRule, error) {
	rule := domain.AccessBlockRule{}
	if existing != nil {
		rule = *existing
	}
	rule.Kind = normalizeAccessBlockKind(firstAccessBlockNonEmptyString(input.Kind, rule.Kind))
	rule.MatchType = normalizeAccessBlockMatchType(firstAccessBlockNonEmptyString(input.MatchType, rule.MatchType))
	rule.Pattern = strings.TrimSpace(firstAccessBlockNonEmptyString(input.Pattern, rule.Pattern))
	if input.Enabled != nil {
		rule.Enabled = *input.Enabled
	} else if existing == nil {
		rule.Enabled = true
	}
	if input.Priority != nil {
		rule.Priority = *input.Priority
	} else if existing == nil {
		rule.Priority = 1000
	}
	rule.Action = normalizeAccessBlockAction(firstAccessBlockNonEmptyString(input.Action, rule.Action))
	if rule.Action == "" {
		rule.Action = AccessBlockActionStatus
	}
	if input.StatusCode != 0 {
		rule.StatusCode = input.StatusCode
	} else if rule.StatusCode == 0 {
		rule.StatusCode = 444
	}
	rule.RedirectURL = strings.TrimSpace(firstAccessBlockNonEmptyString(input.RedirectURL, rule.RedirectURL))
	rule.Note = strings.TrimSpace(firstAccessBlockNonEmptyString(input.Note, rule.Note))
	if err := validateAccessBlockRule(rule); err != nil {
		return domain.AccessBlockRule{}, err
	}
	return rule, nil
}

func validateAccessBlockRule(rule domain.AccessBlockRule) error {
	if rule.Kind == "" || rule.MatchType == "" || strings.TrimSpace(rule.Pattern) == "" {
		return fmt.Errorf("%w: required fields", ErrAccessBlockValidation)
	}
	if len(rule.Pattern) > accessBlockPatternMaxLength {
		return fmt.Errorf("%w: pattern too long", ErrAccessBlockValidation)
	}
	switch rule.Kind {
	case AccessBlockKindIP:
		if rule.MatchType != AccessBlockMatchIP && rule.MatchType != AccessBlockMatchCIDR {
			return fmt.Errorf("%w: invalid ip match type", ErrAccessBlockValidation)
		}
	case AccessBlockKindUA:
		if rule.MatchType != AccessBlockMatchUAContains && rule.MatchType != AccessBlockMatchUARegex {
			return fmt.Errorf("%w: invalid ua match type", ErrAccessBlockValidation)
		}
	default:
		return fmt.Errorf("%w: invalid kind", ErrAccessBlockValidation)
	}
	if _, ok := compileAccessBlockRule(accessBlockRuleDTO(rule)); !ok {
		return fmt.Errorf("%w: invalid pattern", ErrAccessBlockValidation)
	}
	if rule.Action == AccessBlockActionRedirect {
		if !validAccessBlockRedirectURL(rule.RedirectURL) {
			return fmt.Errorf("%w: invalid redirect url", ErrAccessBlockValidation)
		}
		return nil
	}
	if rule.Action != AccessBlockActionStatus || !validAccessBlockStatusCode(rule.StatusCode) {
		return fmt.Errorf("%w: invalid action", ErrAccessBlockValidation)
	}
	return nil
}

func accessBlockRuleMatchesInput(rule domain.AccessBlockRule, input AccessBlockMatchInput) bool {
	compiled, ok := compileAccessBlockRule(accessBlockRuleDTO(rule))
	if !ok {
		return false
	}
	switch rule.MatchType {
	case AccessBlockMatchIP:
		ip, ok := parseAccessBlockAddr(input.IP)
		return ok && ip == compiled.addr
	case AccessBlockMatchCIDR:
		ip, ok := parseAccessBlockAddr(input.IP)
		return ok && compiled.prefix.Contains(ip)
	case AccessBlockMatchUAContains:
		return strings.Contains(strings.ToLower(input.UserAgent), compiled.contains)
	case AccessBlockMatchUARegex:
		return compiled.regex != nil && compiled.regex.MatchString(input.UserAgent)
	default:
		return false
	}
}

func validAccessBlockStatusCode(code int) bool {
	switch code {
	case 403, 404, 410, 429, 444:
		return true
	default:
		return false
	}
}

func validAccessBlockRedirectURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func normalizeAccessBlockKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AccessBlockKindIP:
		return AccessBlockKindIP
	case AccessBlockKindUA, "user_agent", "user-agent":
		return AccessBlockKindUA
	default:
		return ""
	}
}

func normalizeAccessBlockMatchType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AccessBlockMatchIP, "exact_ip":
		return AccessBlockMatchIP
	case AccessBlockMatchCIDR:
		return AccessBlockMatchCIDR
	case AccessBlockMatchUAContains, "contains":
		return AccessBlockMatchUAContains
	case AccessBlockMatchUARegex, "regex":
		return AccessBlockMatchUARegex
	default:
		return ""
	}
}

func normalizeAccessBlockAction(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", AccessBlockActionStatus:
		return AccessBlockActionStatus
	case AccessBlockActionRedirect:
		return AccessBlockActionRedirect
	default:
		return ""
	}
}

func parseAccessBlockAddr(value string) (netip.Addr, bool) {
	ip, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return netip.Addr{}, false
	}
	return ip.Unmap(), true
}

func parseAccessBlockPrefix(value string) (netip.Prefix, bool) {
	prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
	if err != nil {
		return netip.Prefix{}, false
	}
	return prefix.Masked(), true
}

func accessBlockRuleDTO(row domain.AccessBlockRule) AccessBlockRuleDTO {
	return AccessBlockRuleDTO{
		ID:             row.ID,
		ImportSourceID: row.ImportSourceID,
		Kind:           row.Kind,
		MatchType:      row.MatchType,
		Pattern:        row.Pattern,
		Enabled:        row.Enabled,
		Priority:       row.Priority,
		Action:         row.Action,
		StatusCode:     row.StatusCode,
		RedirectURL:    row.RedirectURL,
		Note:           row.Note,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func firstAccessBlockNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

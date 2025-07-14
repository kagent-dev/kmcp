package sanitizer

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Sanitizer removes sensitive information from data structures
type Sanitizer struct {
	patterns []Pattern
}

// Pattern represents a secret detection pattern
type Pattern struct {
	Name        string
	Regex       *regexp.Regexp
	Replacement string
}

// NewSanitizer creates a new sanitizer with default patterns
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		patterns: getDefaultPatterns(),
	}
}

// Sanitize recursively sanitizes data structures, replacing detected secrets
func (s *Sanitizer) Sanitize(data interface{}) interface{} {
	return s.sanitizeValue(reflect.ValueOf(data)).Interface()
}

// sanitizeValue recursively processes different types of values
func (s *Sanitizer) sanitizeValue(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}

	switch v.Kind() {
	case reflect.String:
		return reflect.ValueOf(s.sanitizeString(v.String()))
	case reflect.Map:
		return s.sanitizeMap(v)
	case reflect.Slice, reflect.Array:
		return s.sanitizeSlice(v)
	case reflect.Struct:
		return s.sanitizeStruct(v)
	case reflect.Ptr:
		if v.IsNil() {
			return v
		}
		elem := s.sanitizeValue(v.Elem())
		newPtr := reflect.New(elem.Type())
		newPtr.Elem().Set(elem)
		return newPtr
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		return s.sanitizeValue(v.Elem())
	default:
		// For basic types (int, bool, etc.), return as-is
		return v
	}
}

// sanitizeString applies all patterns to a string value
func (s *Sanitizer) sanitizeString(str string) string {
	for _, pattern := range s.patterns {
		str = pattern.Regex.ReplaceAllString(str, pattern.Replacement)
	}
	return str
}

// sanitizeMap processes map values
func (s *Sanitizer) sanitizeMap(v reflect.Value) reflect.Value {
	if v.IsNil() {
		return v
	}

	newMap := reflect.MakeMap(v.Type())
	for _, key := range v.MapKeys() {
		sanitizedKey := s.sanitizeValue(key)
		sanitizedValue := s.sanitizeValue(v.MapIndex(key))
		newMap.SetMapIndex(sanitizedKey, sanitizedValue)
	}
	return newMap
}

// sanitizeSlice processes slice/array elements
func (s *Sanitizer) sanitizeSlice(v reflect.Value) reflect.Value {
	newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
	for i := 0; i < v.Len(); i++ {
		sanitizedElem := s.sanitizeValue(v.Index(i))
		newSlice.Index(i).Set(sanitizedElem)
	}
	return newSlice
}

// sanitizeStruct processes struct fields
func (s *Sanitizer) sanitizeStruct(v reflect.Value) reflect.Value {
	newStruct := reflect.New(v.Type()).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.CanInterface() {
			sanitizedField := s.sanitizeValue(field)
			if newStruct.Field(i).CanSet() {
				newStruct.Field(i).Set(sanitizedField)
			}
		}
	}

	return newStruct
}

// AddPattern adds a custom pattern to the sanitizer
func (s *Sanitizer) AddPattern(name, pattern, replacement string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern for %s: %w", name, err)
	}

	s.patterns = append(s.patterns, Pattern{
		Name:        name,
		Regex:       regex,
		Replacement: replacement,
	})

	return nil
}

// getDefaultPatterns returns common secret patterns
func getDefaultPatterns() []Pattern {
	patterns := []Pattern{
		// GitHub tokens
		{
			Name:        "GitHub Personal Access Token",
			Regex:       regexp.MustCompile(`ghp_[A-Za-z0-9]{36}`),
			Replacement: "[REDACTED-GITHUB-TOKEN]",
		},
		{
			Name:        "GitHub App Token",
			Regex:       regexp.MustCompile(`ghs_[A-Za-z0-9]{36}`),
			Replacement: "[REDACTED-GITHUB-APP-TOKEN]",
		},
		{
			Name:        "GitHub OAuth Token",
			Regex:       regexp.MustCompile(`gho_[A-Za-z0-9]{36}`),
			Replacement: "[REDACTED-GITHUB-OAUTH-TOKEN]",
		},
		{
			Name:        "GitHub User-to-Server Token",
			Regex:       regexp.MustCompile(`ghu_[A-Za-z0-9]{36}`),
			Replacement: "[REDACTED-GITHUB-USER-TOKEN]",
		},
		{
			Name:        "GitHub Server-to-Server Token",
			Regex:       regexp.MustCompile(`ghr_[A-Za-z0-9]{36}`),
			Replacement: "[REDACTED-GITHUB-SERVER-TOKEN]",
		},
		{
			Name:        "GitHub Refresh Token",
			Regex:       regexp.MustCompile(`ghs_[A-Za-z0-9]{36}`),
			Replacement: "[REDACTED-GITHUB-REFRESH-TOKEN]",
		},

		// OpenAI API Keys
		{
			Name:        "OpenAI API Key",
			Regex:       regexp.MustCompile(`sk-[A-Za-z0-9]{48}`),
			Replacement: "[REDACTED-OPENAI-KEY]",
		},

		// Anthropic API Keys
		{
			Name:        "Anthropic API Key",
			Regex:       regexp.MustCompile(`sk-ant-[A-Za-z0-9\-_]{95}`),
			Replacement: "[REDACTED-ANTHROPIC-KEY]",
		},

		// Slack tokens
		{
			Name:        "Slack Bot Token",
			Regex:       regexp.MustCompile(`xoxb-[0-9]+-[0-9]+-[A-Za-z0-9]+`),
			Replacement: "[REDACTED-SLACK-BOT-TOKEN]",
		},
		{
			Name:        "Slack User Token",
			Regex:       regexp.MustCompile(`xoxp-[0-9]+-[0-9]+-[0-9]+-[A-Za-z0-9]+`),
			Replacement: "[REDACTED-SLACK-USER-TOKEN]",
		},
		{
			Name:        "Slack App Token",
			Regex:       regexp.MustCompile(`xapp-[0-9]+-[A-Za-z0-9]+-[A-Za-z0-9]+`),
			Replacement: "[REDACTED-SLACK-APP-TOKEN]",
		},

		// AWS tokens
		{
			Name:        "AWS Access Key",
			Regex:       regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			Replacement: "[REDACTED-AWS-ACCESS-KEY]",
		},
		{
			Name:        "AWS Secret Key",
			Regex:       regexp.MustCompile(`[A-Za-z0-9/+=]{40}`),
			Replacement: "[REDACTED-AWS-SECRET-KEY]",
		},

		// Google API Keys
		{
			Name:        "Google API Key",
			Regex:       regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),
			Replacement: "[REDACTED-GOOGLE-API-KEY]",
		},

		// JWT tokens
		{
			Name:        "JWT Token",
			Regex:       regexp.MustCompile(`eyJ[A-Za-z0-9\-_]+\.eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+`),
			Replacement: "[REDACTED-JWT-TOKEN]",
		},

		// Bearer tokens (generic)
		{
			Name:        "Bearer Token",
			Regex:       regexp.MustCompile(`Bearer\s+[A-Za-z0-9\-_]+`),
			Replacement: "Bearer [REDACTED-TOKEN]",
		},

		// Database URLs
		{
			Name:        "PostgreSQL URL",
			Regex:       regexp.MustCompile(`postgresql://[^:]+:[^@]+@[^/]+/[^?\s]+`),
			Replacement: "postgresql://[USER]:[REDACTED]@[HOST]/[DB]",
		},
		{
			Name:        "MySQL URL",
			Regex:       regexp.MustCompile(`mysql://[^:]+:[^@]+@[^/]+/[^?\s]+`),
			Replacement: "mysql://[USER]:[REDACTED]@[HOST]/[DB]",
		},
		{
			Name:        "MongoDB URL",
			Regex:       regexp.MustCompile(`mongodb://[^:]+:[^@]+@[^/]+/[^?\s]+`),
			Replacement: "mongodb://[USER]:[REDACTED]@[HOST]/[DB]",
		},
		{
			Name:        "Redis URL",
			Regex:       regexp.MustCompile(`redis://[^:]+:[^@]+@[^/]+/[^?\s]+`),
			Replacement: "redis://[USER]:[REDACTED]@[HOST]/[DB]",
		},

		// Generic password patterns
		{
			Name:        "Password in URL",
			Regex:       regexp.MustCompile(`://[^:]+:([^@]+)@`),
			Replacement: "://[USER]:[REDACTED]@",
		},

		// Credit card numbers (basic pattern)
		{
			Name:        "Credit Card Number",
			Regex:       regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`),
			Replacement: "[REDACTED-CREDIT-CARD]",
		},

		// Email addresses (in some contexts)
		{
			Name:        "Email in Auth Context",
			Regex:       regexp.MustCompile(`"email":\s*"[^"]+"`),
			Replacement: `"email": "[REDACTED-EMAIL]"`,
		},

		// SSH private keys
		{
			Name:        "SSH Private Key",
			Regex:       regexp.MustCompile(`-----BEGIN [A-Z ]+PRIVATE KEY-----[^-]+-----END [A-Z ]+PRIVATE KEY-----`),
			Replacement: "[REDACTED-SSH-PRIVATE-KEY]",
		},

		// Generic high-entropy strings (potential secrets)
		{
			Name:        "High Entropy String",
			Regex:       regexp.MustCompile(`\b[A-Za-z0-9+/]{32,}={0,2}\b`),
			Replacement: "[REDACTED-HIGH-ENTROPY-STRING]",
		},
	}

	return patterns
}

// SanitizeString is a convenience function for sanitizing a single string
func SanitizeString(input string) string {
	sanitizer := NewSanitizer()
	return sanitizer.sanitizeString(input)
}

// IsLikelySecret performs a heuristic check if a string looks like a secret
func IsLikelySecret(input string) bool {
	sanitizer := NewSanitizer()
	sanitized := sanitizer.sanitizeString(input)

	// If the sanitized version is different, it likely contained a secret
	return sanitized != input
}

// GetRedactedFieldNames returns common field names that often contain secrets
func GetRedactedFieldNames() []string {
	return []string{
		"password", "pass", "passwd",
		"secret", "key", "token", "auth",
		"credential", "cred", "authorization",
		"api_key", "apikey", "access_token",
		"refresh_token", "client_secret",
		"private_key", "cert", "certificate",
		"webhook", "webhook_url",
	}
}

// ShouldRedactField checks if a field name suggests it contains sensitive data
func ShouldRedactField(fieldName string) bool {
	fieldName = strings.ToLower(fieldName)
	redactedFields := GetRedactedFieldNames()

	for _, pattern := range redactedFields {
		if strings.Contains(fieldName, pattern) {
			return true
		}
	}

	return false
}

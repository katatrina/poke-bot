package service

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

// Sanitize input to prevent injection attacks and ensure data safety
// Uses html.EscapeString for XSS prevention and includes prompt injection detection

var (
	// Common prompt injection patterns
	promptInjectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)ignore\s+(previous|above|all|prior)\s+(instructions?|prompts?|rules?)`),
		regexp.MustCompile(`(?i)disregard\s+(previous|above|all|prior)\s+(instructions?|prompts?|rules?)`),
		regexp.MustCompile(`(?i)forget\s+(previous|above|all|prior)\s+(instructions?|prompts?|rules?)`),
		regexp.MustCompile(`(?i)you\s+are\s+(now|actually)\s+a`),
		regexp.MustCompile(`(?i)new\s+instructions?:`),
		regexp.MustCompile(`(?i)system\s*:\s*`),
		regexp.MustCompile(`(?i)override\s+(previous|above|all|prior)`),
		regexp.MustCompile(`(?i)act\s+as\s+if\s+you\s+are`),
	}

	// Suspicious control characters (except newlines and tabs)
	controlCharPattern = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
)

// SanitizeInput performs input sanitization using html.EscapeString
func SanitizeInput(input string) string {
	// 1. Trim whitespace
	cleaned := strings.TrimSpace(input)

	// 2. Remove control characters (except newlines and tabs)
	cleaned = controlCharPattern.ReplaceAllString(cleaned, "")

	// 3. Escape HTML entities to prevent XSS
	cleaned = html.EscapeString(cleaned)

	// 4. Normalize excessive whitespace
	cleaned = normalizeWhitespace(cleaned)

	// 5. Limit consecutive newlines
	cleaned = limitConsecutiveNewlines(cleaned, 3)

	return cleaned
}

// DetectPromptInjection checks for common prompt injection patterns
func DetectPromptInjection(input string) bool {
	lowerInput := strings.ToLower(input)

	// Check against known patterns
	for _, pattern := range promptInjectionPatterns {
		if pattern.MatchString(lowerInput) {
			return true
		}
	}

	// Check for excessive repetition (a common prompt injection technique)
	if hasExcessiveRepetition(input) {
		return true
	}

	return false
}

// normalizeWhitespace replaces multiple spaces with a single space
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	spacePattern := regexp.MustCompile(`[ \t]+`)
	return spacePattern.ReplaceAllString(s, " ")
}

// limitConsecutiveNewlines limits the number of consecutive newlines
func limitConsecutiveNewlines(s string, max int) string {
	pattern := regexp.MustCompile(`\n{` + strings.Repeat("", max) + `,}`)
	replacement := strings.Repeat("\n", max)
	return pattern.ReplaceAllString(s, replacement)
}

// hasExcessiveRepetition detects if input has suspicious repetition patterns
func hasExcessiveRepetition(s string) bool {
	if len(s) < 20 {
		return false
	}

	// Check for repeated characters (more than 50 of the same character)
	charCount := make(map[rune]int)
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			charCount[r]++
			if charCount[r] > 50 {
				return true
			}
		}
	}

	// Check for repeated short sequences
	words := strings.Fields(s)
	if len(words) > 10 {
		wordCount := make(map[string]int)
		for _, word := range words {
			wordCount[strings.ToLower(word)]++
			// If same word appears more than 30% of total words, it's suspicious
			if float64(wordCount[strings.ToLower(word)])/float64(len(words)) > 0.3 {
				return true
			}
		}
	}

	return false
}

// ValidateMessageLength checks if message length is within acceptable bounds
func ValidateMessageLength(message string, maxLength int) error {
	if len(message) == 0 {
		return ErrEmptyMessage
	}
	if len(message) > maxLength {
		return ErrMessageTooLong
	}
	return nil
}

// Custom errors
var (
	ErrEmptyMessage       = newValidationError("message cannot be empty")
	ErrMessageTooLong     = newValidationError("message exceeds maximum length")
	ErrPromptInjection    = newValidationError("message contains suspicious patterns")
	ErrInvalidCharacters  = newValidationError("message contains invalid characters")
)

type validationError struct {
	message string
}

func newValidationError(msg string) error {
	return &validationError{message: msg}
}

func (e *validationError) Error() string {
	return e.message
}

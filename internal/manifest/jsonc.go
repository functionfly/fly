package manifest

import (
	"bytes"
	"strings"
	"unicode"
)

// StripComments removes JSONC comments from input and returns clean JSON.
// It handles:
// - Single-line comments: // comment
// - Multi-line comments: /* comment */
// Comments inside strings are preserved (not stripped)
func StripComments(input string) string {
	// Use a state machine to track whether we're inside a string
	var result strings.Builder
	// Estimate capacity: input size minus comment size
	result.Grow(len(input))

	i := 0
	inString := false
	escapeNext := false
	lastWasSpace := false

	for i < len(input) {
		ch := rune(input[i])

		// Handle escape sequences
		if escapeNext {
			result.WriteRune(ch)
			escapeNext = false
			lastWasSpace = false
			i++
			continue
		}

		// Handle escape character
		if ch == '\\' && inString {
			result.WriteRune(ch)
			escapeNext = true
			lastWasSpace = false
			i++
			continue
		}

		// Toggle string state
		if ch == '"' {
			inString = !inString
			result.WriteRune(ch)
			lastWasSpace = false
			i++
			continue
		}

		// If inside a string, just copy the character
		if inString {
			result.WriteRune(ch)
			lastWasSpace = false
			i++
			continue
		}

		// Handle single-line comment: //
		if i+1 < len(input) && ch == '/' && rune(input[i+1]) == '/' {
			// Skip until end of line or end of input
			for i < len(input) && input[i] != '\n' && input[i] != '\r' {
				i++
			}
			// Skip the newline character and add a space if needed
			if i < len(input) {
				i++ // skip the newline
				// Add space only if we have content and last wasn't a space
				if result.Len() > 0 && !lastWasSpace {
					result.WriteRune(' ')
					lastWasSpace = true
				}
			}
			continue
		}

		// Handle multi-line comment: /* */
		if i+1 < len(input) && ch == '/' && rune(input[i+1]) == '*' {
			// Skip until end of comment
			i += 2 // skip /*
			for i+1 < len(input) {
				if input[i] == '*' && i+1 < len(input) && input[i+1] == '/' {
					i += 2 // skip */
					break
				}
				i++
			}
			// Add space only if we have content and last wasn't a space
			if result.Len() > 0 && !lastWasSpace {
				result.WriteRune(' ')
				lastWasSpace = true
			}
			continue
		}

		// Handle whitespace
		if unicode.IsSpace(ch) {
			// Collapse multiple whitespace into single space
			if !lastWasSpace && result.Len() > 0 {
				result.WriteRune(' ')
				lastWasSpace = true
			}
			i++
			continue
		}

		// Regular character - copy it
		result.WriteRune(ch)
		lastWasSpace = false
		i++
	}

	return result.String()
}

// StripCommentsBytes is the byte slice version of StripComments
func StripCommentsBytes(input []byte) []byte {
	return []byte(StripComments(string(input)))
}

// IsValidJSON checks if the input is valid JSON (after comment stripping)
func IsValidJSON(input string) bool {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) == 0 {
		return false
	}
	cleaned := StripComments(trimmed)
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) == 0 {
		return false
	}
	return (cleaned[0] == '{' && cleaned[len(cleaned)-1] == '}') ||
		(cleaned[0] == '[' && cleaned[len(cleaned)-1] == ']')
}

// FormatJSONC formats JSON with optional comments (JSONC) for display
// This is a no-op since we don't prettify comments, just returns formatted JSON
func FormatJSONC(input string, indent string) (string, error) {
	cleaned := StripComments(input)

	var buf bytes.Buffer
	// Simple pretty print - indent with spaces
	level := 0
	inString := false

	for i := 0; i < len(cleaned); i++ {
		ch := cleaned[i]

		if ch == '"' && (i == 0 || cleaned[i-1] != '\\') {
			inString = !inString
			buf.WriteByte(ch)
			continue
		}

		if inString {
			buf.WriteByte(ch)
			continue
		}

		switch ch {
		case '{', '[':
			buf.WriteByte(ch)
			buf.WriteString("\n")
			level++
			buf.WriteString(indent)
		case '}', ']':
			buf.WriteString("\n")
			level--
			buf.WriteString(indent)
			buf.WriteByte(ch)
		case ',':
			buf.WriteByte(ch)
			buf.WriteString("\n")
			buf.WriteString(indent)
		case ':':
			buf.WriteString(": ")
		default:
			if !unicode.IsSpace(rune(ch)) {
				buf.WriteByte(ch)
			}
		}
	}

	return buf.String(), nil
}

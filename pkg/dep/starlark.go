package dep

import (
	"strings"
)

// Minimal, lenient helpers for reading Starlark (Bazel) files.
// These do not evaluate code: they find function calls, keyword arguments and
// string literals, while correctly skipping over strings, comments and nested
// brackets.

type starlarkCall struct {
	name string // The (possibly dotted) function name, e.g. "java_library" or "maven.install".
	args string // The raw text between the call's parentheses.
}

// starlarkCalls returns the calls in src, not including calls nested inside
// the arguments of another call.
func starlarkCalls(src string) []starlarkCall {
	var calls []starlarkCall
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '"' || c == '\'':
			i = skipStarlarkString(src, i)
		case isIdentStart(c) && (i == 0 || !isIdentByte(src[i-1]) && src[i-1] != '.'):
			end := i
			for end < len(src) && (isIdentByte(src[end]) || src[end] == '.') {
				end++
			}
			name := src[i:end]
			j := end
			for j < len(src) && isStarlarkSpace(src[j]) {
				j++
			}
			if j < len(src) && src[j] == '(' {
				closing := matchStarlarkBracket(src, j)
				calls = append(calls, starlarkCall{name: name, args: src[j+1 : closing]})
				i = closing + 1
				continue
			}
			i = end
		default:
			i++
		}
	}
	return calls
}

// splitStarlarkArgs splits a call's arguments by top-level commas.
func splitStarlarkArgs(args string) []string {
	return splitStarlarkTopLevel(args, ',')
}

// splitStarlarkTopLevel splits an expression by a separator that is not
// nested in brackets or strings.
func splitStarlarkTopLevel(expr string, sep byte) []string {
	var parts []string
	start, depth := 0, 0
	for i := 0; i < len(expr); {
		switch c := expr[i]; c {
		case '"', '\'':
			i = skipStarlarkString(expr, i)
			continue
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case sep:
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(expr[start:i]))
				start = i + 1
			}
		}
		i++
	}
	if last := strings.TrimSpace(expr[start:]); last != "" {
		parts = append(parts, last)
	}
	return parts
}

// starlarkKwarg returns the value expression of the keyword argument key.
func starlarkKwarg(args []string, key string) string {
	for _, arg := range args {
		rest, ok := strings.CutPrefix(arg, key)
		if !ok {
			continue
		}
		rest = strings.TrimLeft(rest, " \t\r\n")
		if value, ok := strings.CutPrefix(rest, "="); ok && !strings.HasPrefix(value, "=") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// starlarkAssignment returns the value expression assigned to a top-level
// variable in src, e.g. the list in "ARTIFACTS = [...]" or the whole of
// "ARTIFACTS = BASE + [...]".
func starlarkAssignment(src, name string) string {
	for i := 0; i < len(src); {
		c := src[i]
		if c == '"' || c == '\'' {
			i = skipStarlarkString(src, i)
			continue
		}
		if strings.HasPrefix(src[i:], name) && (i == 0 || src[i-1] == '\n') {
			rest := strings.TrimLeft(src[i+len(name):], " \t")
			if value, ok := strings.CutPrefix(rest, "="); ok && !strings.HasPrefix(value, "=") {
				return strings.TrimSpace(value[:starlarkLineEnd(value)])
			}
		}
		i++
	}
	return ""
}

// starlarkLineEnd returns the index of the first newline in src that is not
// nested in brackets or strings, or the length of src.
func starlarkLineEnd(src string) int {
	depth := 0
	for i := 0; i < len(src); {
		switch src[i] {
		case '"', '\'':
			i = skipStarlarkString(src, i)
			continue
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case '\n':
			if depth <= 0 {
				return i
			}
		}
		i++
	}
	return len(src)
}

// starlarkStrings returns the values of all string literals in src.
// Escape sequences are not interpreted, other than to find the end of strings.
func starlarkStrings(src string) []string {
	var lits []string
	for i := 0; i < len(src); {
		c := src[i]
		if c != '"' && c != '\'' {
			i++
			continue
		}
		end := skipStarlarkString(src, i)
		delim := string(c)
		if strings.HasPrefix(src[i:], strings.Repeat(delim, 3)) {
			delim = strings.Repeat(delim, 3)
		}
		// Skip unterminated strings.
		if q := len(delim); end-q >= i+q && src[end-q:end] == delim {
			lits = append(lits, src[i+q:end-q])
		}
		i = end
	}
	return lits
}

// firstString returns the first string literal in an expression.
func firstString(expr string) string {
	if lits := starlarkStrings(expr); len(lits) > 0 {
		return lits[0]
	}
	return ""
}

// stripStarlarkComments removes "#" comments from src.
func stripStarlarkComments(src string) string {
	if !strings.Contains(src, "#") {
		return src
	}
	var b strings.Builder
	b.Grow(len(src))
	for i := 0; i < len(src); {
		switch c := src[i]; c {
		case '"', '\'':
			end := skipStarlarkString(src, i)
			b.WriteString(src[i:end])
			i = end
		case '#':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// skipStarlarkString returns the index after the string literal starting at i.
// Unterminated strings run to the end of src.
func skipStarlarkString(src string, i int) int {
	quote := src[i]
	delim := string(quote)
	if strings.HasPrefix(src[i:], strings.Repeat(delim, 3)) {
		delim = strings.Repeat(delim, 3)
	}
	for j := i + len(delim); j < len(src); j++ {
		if src[j] == '\\' {
			j++
			continue
		}
		if strings.HasPrefix(src[j:], delim) {
			return j + len(delim)
		}
		if len(delim) == 1 && src[j] == '\n' {
			return j
		}
	}
	return len(src)
}

// matchStarlarkBracket returns the index of the bracket closing the one at
// index open. If it is unclosed, the length of src is returned.
func matchStarlarkBracket(src string, open int) int {
	depth := 0
	for i := open; i < len(src); {
		switch src[i] {
		case '"', '\'':
			i = skipStarlarkString(src, i)
			continue
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return len(src)
}

func isIdentStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isIdentByte(c byte) bool {
	return isIdentStart(c) || c >= '0' && c <= '9'
}

func isStarlarkSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

func isStarlarkIdent(s string) bool {
	if s == "" || !isIdentStart(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isIdentByte(s[i]) {
			return false
		}
	}
	return true
}

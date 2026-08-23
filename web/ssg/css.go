package ssg

import "strings"

// scopeCSS prefixes every plain rule in css with the scope selector and
// recurses into @media/@supports/@layer blocks. Rules that begin with :root
// (or any selector containing :root) are left global as an escape hatch, as
// are @keyframes, @font-face, @import, and @charset at-rules.
//
// When compound is true, simple selectors are also emitted compounded with
// the scope (e.g. .x -> "[scope] .x, [scope].x") so the component's own root
// element, which carries the scope attribute, is styled as well.
func scopeCSS(scope, css string, compound bool) string {
	var out strings.Builder
	rest := css
	for rest != "" {
		rest = skipSpace(rest)
		if rest == "" {
			break
		}
		open := findBrace(rest)
		if open < 0 {
			out.WriteString(rest)
			break
		}
		body, consumed, ok := consumeBlock(rest, open)
		if !ok {
			out.WriteString(rest)
			break
		}
		head := rest[:open]
		rest = rest[consumed:]
		if isAtRule(head) {
			out.WriteString(scopeAtRule(scope, head, body, compound))
			continue
		}
		out.WriteString(scopeRule(scope, head, body, compound))
	}
	return out.String()
}

func scopeRule(scope, head, body string, compound bool) string {
	head = strings.TrimSpace(head)
	if head == "" || strings.Contains(head, ":root") {
		return head + "{" + body + "}"
	}
	var out strings.Builder
	for _, sel := range splitSelectors(head) {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		out.WriteString(scope)
		out.WriteByte(' ')
		out.WriteString(sel)
		out.WriteString(", ")
		if compound && !hasCombinator(sel) {
			out.WriteString(scope)
			out.WriteString(sel)
			out.WriteString(", ")
		}
	}
	res := out.String()
	if strings.HasSuffix(res, ", ") {
		res = res[:len(res)-2]
	}
	return res + "{" + body + "}"
}

func scopeAtRule(scope, head, body string, compound bool) string {
	fields := strings.Fields(head)
	if len(fields) == 0 {
		return head + "{" + body + "}"
	}
	switch fields[0] {
	case "@media", "@supports", "@layer":
		return head + "{" + scopeCSS(scope, body, compound) + "}"
	default:
		return head + "{" + body + "}"
	}
}

// hasCombinator reports whether sel contains a combinator (space, >, +, ~)
// outside of brackets, parentheses, or strings.
func hasCombinator(sel string) bool {
	depth := 0
	inStr := byte(0)
	for i := 0; i < len(sel); i++ {
		c := sel[i]
		if inStr != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = c
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case ' ', '\t', '\n', '>', '+', '~':
			if depth == 0 {
				return true
			}
		}
	}
	return false
}

// splitSelectors splits a comma-separated selector list on top-level commas,
// ignoring commas inside parentheses, brackets, and strings.
func splitSelectors(s string) []string {
	var parts []string
	depth := 0
	inStr := byte(0)
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = c
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// findBrace returns the index of the first top-level '{', ignoring braces
// inside strings, comments, parentheses, and brackets.
func findBrace(s string) int {
	depth := 0
	inStr := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == inStr {
				inStr = 0
			}
			continue
		}
		switch {
		case c == '"' || c == '\'':
			inStr = c
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			if depth > 0 {
				depth--
			}
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			if end := strings.Index(s[i+2:], "*/"); end >= 0 {
				i += 2 + end + 2
			} else {
				return -1
			}
		case c == '{' && depth == 0:
			return i
		}
	}
	return -1
}

// consumeBlock reads a brace-delimited block starting at open, returning the
// inner body text and the number of bytes consumed including the closing brace.
func consumeBlock(s string, open int) (body string, consumed int, ok bool) {
	depth := 0
	inStr := byte(0)
	for i := open; i < len(s); i++ {
		c := s[i]
		if inStr != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = c
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[open+1 : i], i + 1, true
			}
		case '/':
			if i+1 < len(s) && s[i+1] == '*' {
				if end := strings.Index(s[i+2:], "*/"); end >= 0 {
					i += 2 + end + 2
				} else {
					return "", 0, false
				}
			}
		}
	}
	return "", 0, false
}

// skipSpace skips leading whitespace and comments.
func skipSpace(s string) string {
	for i := 0; i < len(s); {
		switch {
		case s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r':
			i++
		case s[i] == '/' && i+1 < len(s) && s[i+1] == '*':
			if end := strings.Index(s[i+2:], "*/"); end >= 0 {
				i += 2 + end + 2
			} else {
				return ""
			}
		default:
			return s[i:]
		}
	}
	return ""
}

func isAtRule(head string) bool {
	return strings.HasPrefix(strings.TrimSpace(head), "@")
}

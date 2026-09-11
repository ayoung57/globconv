// Package convert translates between .gitignore glob syntax and rsync
// filter-rule glob syntax, line by line.
package convert

import (
	"fmt"
	"regexp"
	"strings"
)

// Options controls how a conversion handles patterns that don't have a
// clean equivalent in the target format.
type Options struct {
	// Lenient allows the converter to make a best-effort translation of
	// patterns that would otherwise be rejected, emitting a Warning for
	// each one instead of stopping. Without it, any such pattern is a
	// hard error.
	Lenient bool
}

// Warning describes a pattern that was rewritten rather than translated
// exactly, because it had no direct equivalent in the target format.
type Warning struct {
	Line    int
	Message string
}

var runOfStars = regexp.MustCompile(`\*{2,}`)

// GitignoreToRsync converts .gitignore pattern lines into rsync filter
// rules (the "+ pattern" / "- pattern" form used by --filter-file and
// merge files).
func GitignoreToRsync(lines []string, opts Options) ([]string, []Warning, error) {
	out := make([]string, 0, len(lines))
	var warnings []Warning

	for i, raw := range lines {
		lineNo := i + 1

		if strings.TrimSpace(raw) == "" {
			out = append(out, "")
			continue
		}
		if raw[0] == '#' {
			out = append(out, raw)
			continue
		}

		work := raw
		negated := false
		switch {
		case strings.HasPrefix(work, "\\#"), strings.HasPrefix(work, "\\!"):
			work = work[1:]
		case strings.HasPrefix(work, "!"):
			negated = true
			work = work[1:]
		}

		work = trimTrailingGitSpaces(work)

		fixed, changes, err := fixDoubleStar(work, opts.Lenient)
		if err != nil {
			return nil, warnings, fmt.Errorf("line %d: %w", lineNo, err)
		}
		for _, c := range changes {
			warnings = append(warnings, Warning{Line: lineNo, Message: c})
		}

		prefix := "- "
		if negated {
			prefix = "+ "
		}
		out = append(out, prefix+fixed)
	}

	return out, warnings, nil
}

// RsyncToGitignore converts rsync filter rules ("+ pattern" / "- pattern")
// into .gitignore pattern lines.
func RsyncToGitignore(lines []string, opts Options) ([]string, []Warning, error) {
	out := make([]string, 0, len(lines))
	var warnings []Warning

	for i, raw := range lines {
		lineNo := i + 1

		if strings.TrimSpace(raw) == "" {
			out = append(out, "")
			continue
		}
		if raw[0] == '#' {
			out = append(out, raw)
			continue
		}
		if raw[0] == ';' {
			out = append(out, "#"+raw[1:])
			continue
		}

		var included bool
		var pattern string
		switch {
		case strings.HasPrefix(raw, "+ "):
			included, pattern = true, raw[2:]
		case strings.HasPrefix(raw, "- "):
			included, pattern = false, raw[2:]
		default:
			if !opts.Lenient {
				return nil, warnings, fmt.Errorf("line %d: rule %q has no \"+ \" or \"- \" prefix", lineNo, raw)
			}
			warnings = append(warnings, Warning{
				Line:    lineNo,
				Message: fmt.Sprintf("rule %q has no +/- prefix, treating it as an exclude", raw),
			})
			included, pattern = false, raw
		}

		fixed, changes, err := fixDoubleStar(pattern, opts.Lenient)
		if err != nil {
			return nil, warnings, fmt.Errorf("line %d: %w", lineNo, err)
		}
		for _, c := range changes {
			warnings = append(warnings, Warning{Line: lineNo, Message: c})
		}

		if len(fixed) > 0 && (fixed[0] == '#' || fixed[0] == '!') {
			fixed = "\\" + fixed
		}

		if included {
			out = append(out, "!"+fixed)
		} else {
			out = append(out, fixed)
		}
	}

	return out, warnings, nil
}

// fixDoubleStar checks that every "**" in pattern occupies a whole path
// segment on its own ("**/", "/**", "/**/" or the entire pattern), which is
// the only form both gitignore and rsync agree on. A "**" glued to other
// characters in the same segment (e.g. "a**b") is well defined in neither
// spec the same way, so in lenient mode it's collapsed to a single "*".
func fixDoubleStar(pattern string, lenient bool) (string, []string, error) {
	segments := strings.Split(pattern, "/")
	var changes []string

	for i, seg := range segments {
		if seg == "" || seg == "**" || !strings.Contains(seg, "**") {
			continue
		}
		if !lenient {
			return "", nil, fmt.Errorf("segment %q uses \"**\" outside of its own path segment", seg)
		}
		collapsed := runOfStars.ReplaceAllString(seg, "*")
		changes = append(changes, fmt.Sprintf("collapsed %q to %q (\"**\" only has special meaning as a whole path segment)", seg, collapsed))
		segments[i] = collapsed
	}

	return strings.Join(segments, "/"), changes, nil
}

// trimTrailingGitSpaces applies gitignore's rule that trailing spaces are
// stripped from a pattern unless the last one is escaped with a backslash.
func trimTrailingGitSpaces(s string) string {
	for len(s) > 0 && s[len(s)-1] == ' ' {
		if len(s) >= 2 && s[len(s)-2] == '\\' {
			s = s[:len(s)-2] + " "
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

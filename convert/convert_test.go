package convert

import (
	"reflect"
	"testing"
)

func TestGitignoreToRsync_Negation(t *testing.T) {
	in := []string{"*.log", "!important.log"}
	want := []string{"- *.log", "+ important.log"}

	got, warnings, err := GitignoreToRsync(in, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestGitignoreToRsync_EscapedLeadingChars(t *testing.T) {
	in := []string{`\#not-a-comment`, `\!not-negated`}
	want := []string{"- #not-a-comment", "- !not-negated"}

	got, _, err := GitignoreToRsync(in, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestGitignoreToRsync_CommentsAndBlankLines(t *testing.T) {
	in := []string{"# a comment", "", "*.tmp"}
	want := []string{"# a comment", "", "- *.tmp"}

	got, _, err := GitignoreToRsync(in, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestGitignoreToRsync_DoubleStarValidPlacements(t *testing.T) {
	in := []string{"**/foo", "foo/**", "foo/**/bar", "**"}
	want := []string{"- **/foo", "- foo/**", "- foo/**/bar", "- **"}

	got, warnings, err := GitignoreToRsync(in, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings for valid \"**\" placement: %v", warnings)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestGitignoreToRsync_DoubleStarGluedIsStrictError(t *testing.T) {
	_, _, err := GitignoreToRsync([]string{"src/a**b/*.go"}, Options{})
	if err == nil {
		t.Fatal("expected an error for \"**\" glued to other characters, got none")
	}
}

func TestGitignoreToRsync_DoubleStarGluedIsLenientWarning(t *testing.T) {
	got, warnings, err := GitignoreToRsync([]string{"src/a**b/*.go"}, Options{Lenient: true})
	if err != nil {
		t.Fatalf("unexpected error in lenient mode: %v", err)
	}
	want := []string{"- src/a*b/*.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected exactly one warning, got %v", warnings)
	}
	if warnings[0].Line != 1 {
		t.Errorf("warning line = %d, want 1", warnings[0].Line)
	}
}

func TestRsyncToGitignore_Negation(t *testing.T) {
	in := []string{"- *.log", "+ important.log"}
	want := []string{"*.log", "!important.log"}

	got, _, err := RsyncToGitignore(in, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRsyncToGitignore_CommentsAndBlankLines(t *testing.T) {
	in := []string{"# a comment", "; a semicolon comment", "", "- *.tmp"}
	want := []string{"# a comment", "# a semicolon comment", "", "*.tmp"}

	got, _, err := RsyncToGitignore(in, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRsyncToGitignore_ReEscapesLeadingSpecialChars(t *testing.T) {
	in := []string{"- #tag", "+ !bang"}
	want := []string{`\#tag`, `!\!bang`}

	got, _, err := RsyncToGitignore(in, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRsyncToGitignore_MissingPrefixIsStrictError(t *testing.T) {
	_, _, err := RsyncToGitignore([]string{"*.log"}, Options{})
	if err == nil {
		t.Fatal("expected an error for a rule with no +/- prefix, got none")
	}
}

func TestRsyncToGitignore_MissingPrefixIsLenientWarning(t *testing.T) {
	got, warnings, err := RsyncToGitignore([]string{"*.log"}, Options{Lenient: true})
	if err != nil {
		t.Fatalf("unexpected error in lenient mode: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"*.log"}) {
		t.Errorf("got %v, want [*.log]", got)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected exactly one warning, got %v", warnings)
	}
}

func TestRsyncToGitignore_DoubleStarGluedIsStrictError(t *testing.T) {
	_, _, err := RsyncToGitignore([]string{"- src/a**b/*.go"}, Options{})
	if err == nil {
		t.Fatal("expected an error for \"**\" glued to other characters, got none")
	}
}

func TestFixDoubleStar(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		lenient bool
		want    string
		wantErr bool
		wantLen int // expected number of changes
	}{
		{name: "lone segment", pattern: "**", want: "**"},
		{name: "leading", pattern: "**/foo", want: "**/foo"},
		{name: "trailing", pattern: "foo/**", want: "foo/**"},
		{name: "middle", pattern: "foo/**/bar", want: "foo/**/bar"},
		{name: "no stars", pattern: "foo/bar", want: "foo/bar"},
		{name: "glued strict", pattern: "a**b", wantErr: true},
		{name: "glued lenient", pattern: "a**b", lenient: true, want: "a*b", wantLen: 1},
		{name: "run of stars lenient", pattern: "a***b", lenient: true, want: "a*b", wantLen: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changes, err := fixDoubleStar(tt.pattern, tt.lenient)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if len(changes) != tt.wantLen {
				t.Errorf("got %d changes, want %d", len(changes), tt.wantLen)
			}
		})
	}
}

func TestFixDoubleStar_StarsInsideBracketsAreLiteral(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
	}{
		{name: "whole segment is a class of two stars", pattern: "a/[**]/b"},
		{name: "class glued to the rest of the segment", pattern: "a[**]b"},
		{name: "negated class with !", pattern: "a[!**]b"},
		{name: "negated class with ^", pattern: "a[^**]b"},
		{name: "literal ] as first class member", pattern: "a[]**]b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changes, err := fixDoubleStar(tt.pattern, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.pattern {
				t.Errorf("got %q, want %q unchanged", got, tt.pattern)
			}
			if len(changes) != 0 {
				t.Errorf("unexpected changes: %v", changes)
			}
		})
	}
}

func TestFixDoubleStar_GluedStarOutsideBracketStillErrors(t *testing.T) {
	_, _, err := fixDoubleStar("a**[b]", false)
	if err == nil {
		t.Fatal("expected an error for \"**\" glued outside a bracket expression, got none")
	}

	got, changes, err := fixDoubleStar("a**[b]", true)
	if err != nil {
		t.Fatalf("unexpected error in lenient mode: %v", err)
	}
	want := "a*[b]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if len(changes) != 1 {
		t.Fatalf("expected exactly one change, got %v", changes)
	}
}

func TestFixDoubleStar_UnterminatedBracketIsLiteral(t *testing.T) {
	// No closing "]", so the "[" is just a literal character and the "**"
	// after it is a real glued double-star.
	_, _, err := fixDoubleStar("a[**b", false)
	if err == nil {
		t.Fatal("expected an error, got none")
	}
}

func TestGitignoreToRsync_BracketExpressionsPassThrough(t *testing.T) {
	in := []string{"file[**].txt", "[!abc]/dir", "[^0-9]*.go"}
	want := []string{"- file[**].txt", "- [!abc]/dir", "- [^0-9]*.go"}

	got, warnings, err := GitignoreToRsync(in, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTrimTrailingGitSpaces(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"foo", "foo"},
		{"foo  ", "foo"},
		{`foo\ `, "foo "},
		{`foo\  `, "foo "},
		{"", ""},
	}

	for _, tt := range tests {
		if got := trimTrailingGitSpaces(tt.in); got != tt.want {
			t.Errorf("trimTrailingGitSpaces(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	in := []string{
		"# build output",
		"/dist/",
		"*.log",
		"!important.log",
		"**/node_modules",
	}

	rsync, _, err := GitignoreToRsync(in, Options{})
	if err != nil {
		t.Fatalf("GitignoreToRsync: %v", err)
	}

	back, _, err := RsyncToGitignore(rsync, Options{})
	if err != nil {
		t.Fatalf("RsyncToGitignore: %v", err)
	}

	if !reflect.DeepEqual(back, in) {
		t.Errorf("round trip mismatch:\ngot  %v\nwant %v", back, in)
	}
}

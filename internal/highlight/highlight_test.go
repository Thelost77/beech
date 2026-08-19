package highlight

import (
	"strings"
	"testing"
)

func TestParseSupportedMarkdown(t *testing.T) {
	input := "[ ] Review **storage** and `Save` with [issue](https://example.com) #projekt"
	spans := Parse(input)
	if joined := join(spans); joined != input {
		t.Fatalf("joined = %q", joined)
	}
	for _, role := range []Role{TaskPending, Strong, Code, Link, Tag, Syntax} {
		if !hasRole(spans, role) {
			t.Errorf("missing role %v in %#v", role, spans)
		}
	}
}

func TestCompletedTaskAndUnicodeTag(t *testing.T) {
	spans := Parse("[x] Gotowe #ważne")
	if !hasRole(spans, TaskDone) || !hasRole(spans, Tag) {
		t.Fatalf("spans = %#v", spans)
	}
}

func TestMalformedAndEscapedSyntaxStaysPlain(t *testing.T) {
	for _, input := range []string{"Broken **strong", "Broken [link](", `Escaped \**plain**`} {
		spans := Parse(input)
		if joined := join(spans); joined != input {
			t.Fatalf("%q joined as %q", input, joined)
		}
		if strings.HasPrefix(input, "Escaped") && hasRole(spans, Strong) {
			t.Fatalf("escaped syntax was highlighted: %#v", spans)
		}
	}
}

func join(spans []Span) string {
	var result strings.Builder
	for _, span := range spans {
		result.WriteString(span.Text)
	}
	return result.String()
}

func hasRole(spans []Span, role Role) bool {
	for _, span := range spans {
		if span.Role == role {
			return true
		}
	}
	return false
}

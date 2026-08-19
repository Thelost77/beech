package outline

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarkdownRoundTripPreservesTabsAndCollapsedState(t *testing.T) {
	input := []byte("<!-- beech:outline v1 -->\r\n\r\n- Project\r\n\t- Research <!-- beech:collapsed -->\r\n\t\t- Read tools\r\n\t- Implementation\r\n")
	parsed, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	root := parsed.Document.Roots()[0]
	research := parsed.Document.Children(root)[0]
	if !parsed.Collapsed[research] {
		t.Fatal("Research was not restored as collapsed")
	}
	if parsed.Document.Text(research) != "Research" {
		t.Fatalf("marker leaked into node text: %q", parsed.Document.Text(research))
	}
	if parsed.Style.Newline != "\r\n" || !parsed.Style.FinalNewline {
		t.Fatalf("style = %#v", parsed.Style)
	}
	output, err := Serialize(parsed.Document, parsed.Collapsed, parsed.Style)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output, input) {
		t.Fatalf("output:\n%q\nwant:\n%q", output, input)
	}
}

func TestMarkdownWithoutMarkerImportsAndSerializesAsNative(t *testing.T) {
	parsed, err := Parse([]byte("- Root\n\t- Child\n"))
	if err != nil {
		t.Fatal(err)
	}
	output, err := Serialize(parsed.Document, parsed.Collapsed, parsed.Style)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(output), formatMarker+"\n\n- Root") {
		t.Fatalf("output = %q", output)
	}
}

func TestMarkdownTaskSyntaxRemainsNodeText(t *testing.T) {
	parsed, err := Parse([]byte("- Project\n\t- [ ] Pending\n\t- [x] Done\n"))
	if err != nil {
		t.Fatal(err)
	}
	children := parsed.Document.Children(parsed.Document.Roots()[0])
	if got := parsed.Document.Text(children[0]); got != "[ ] Pending" {
		t.Fatalf("pending text = %q", got)
	}
	if got := parsed.Document.Text(children[1]); got != "[x] Done" {
		t.Fatalf("done text = %q", got)
	}
}

func TestMarkdownRejectsSpaceIndentation(t *testing.T) {
	_, err := Parse([]byte("- Root\n  - Child\n"))
	if err == nil || !strings.Contains(err.Error(), "use tabs") {
		t.Fatalf("error = %v", err)
	}
}

func TestMarkdownRejectsReservedMetadataInNodeText(t *testing.T) {
	parsed, err := Parse([]byte("- Root\n"))
	if err != nil {
		t.Fatal(err)
	}
	root := parsed.Document.Roots()[0]
	parsed.Document.Rename(root, "Literal <!-- beech:collapsed -->")
	_, err = Serialize(parsed.Document, nil, parsed.Style)
	if err == nil || !strings.Contains(err.Error(), "reserved Beech metadata") {
		t.Fatalf("error = %v", err)
	}
}

func TestMarkdownRejectsUnsupportedContent(t *testing.T) {
	_, err := Parse([]byte("# Heading\n\nParagraph\n"))
	if err == nil || !strings.Contains(err.Error(), "expected a Markdown list item") {
		t.Fatalf("error = %v", err)
	}
}

func TestMarkdownRejectsSkippedDepth(t *testing.T) {
	_, err := Parse([]byte("- Root\n\t\t- Grandchild\n"))
	if err == nil || !strings.Contains(err.Error(), "skips a tree level") {
		t.Fatalf("error = %v", err)
	}
}

func TestImportHMMAcceptsTabsSpacesAndMultipleRoots(t *testing.T) {
	parsed, err := ImportHMM([]byte("One\n  Child\nTwo\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(parsed.Document.Roots()); got != 2 {
		t.Fatalf("roots = %d", got)
	}
	output, err := Serialize(parsed.Document, parsed.Collapsed, parsed.Style)
	if err != nil {
		t.Fatal(err)
	}
	want := "<!-- beech:outline v1 -->\n\n- One\n\t- Child\n- Two\n"
	if string(output) != want {
		t.Fatalf("converted output = %q, want %q", output, want)
	}
}

func TestImportHMMRejectsUnknownDedent(t *testing.T) {
	_, err := ImportHMM([]byte("Root\n  A\n    B\n   C\n"))
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("error = %v", err)
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte("<!-- beech:outline v1 -->\n\n- Root\n\t- Child\n"))
	f.Add([]byte("- Root\r\n\t- Child <!-- beech:collapsed -->\r\n\t\t- Detail\r\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		parsed, err := Parse(data)
		if err != nil {
			return
		}
		serialized, err := Serialize(parsed.Document, parsed.Collapsed, parsed.Style)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse(serialized); err != nil {
			t.Fatalf("serialized document does not parse: %v", err)
		}
	})
}

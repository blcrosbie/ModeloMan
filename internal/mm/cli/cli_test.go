package cli

import (
	"strings"
	"testing"
)

func TestResolvePromptInputWithReaderPrefersInlinePrompt(t *testing.T) {
	prompt, err := resolvePromptInputWithReader("ship the fix", "", strings.NewReader("ignored"), false)
	if err != nil {
		t.Fatalf("resolve prompt: %v", err)
	}
	if prompt != "ship the fix" {
		t.Fatalf("unexpected prompt %q", prompt)
	}
}

func TestResolvePromptInputWithReaderReadsFullStdin(t *testing.T) {
	prompt, err := resolvePromptInputWithReader("", "", strings.NewReader("line one\nline two\n"), true)
	if err != nil {
		t.Fatalf("resolve prompt from stdin: %v", err)
	}
	if prompt != "line one\nline two" {
		t.Fatalf("unexpected prompt %q", prompt)
	}
}

func TestResolvePromptInputWithReaderRejectsConflictingSources(t *testing.T) {
	_, err := resolvePromptInputWithReader("inline", "prompt.md", strings.NewReader(""), false)
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestResolvePromptInputWithReaderRejectsMissingPrompt(t *testing.T) {
	_, err := resolvePromptInputWithReader("", "", strings.NewReader(""), false)
	if err == nil {
		t.Fatal("expected missing prompt error")
	}
}

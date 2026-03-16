package orchestrate

import "testing"

func TestExtractPRURL(t *testing.T) {
	output := "Creating pull request for feature/F054 into dev in fall-out-bug/sdp_lab\nhttps://github.com/fall-out-bug/sdp_lab/pull/123\n"
	got := extractPRURL(output)
	want := "https://github.com/fall-out-bug/sdp_lab/pull/123"
	if got != want {
		t.Fatalf("extractPRURL() = %q, want %q", got, want)
	}
}

func TestExtractPRURLEmpty(t *testing.T) {
	if got := extractPRURL("no url here"); got != "" {
		t.Fatalf("extractPRURL() = %q, want empty", got)
	}
}

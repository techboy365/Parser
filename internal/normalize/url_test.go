package normalize

import "testing"

func TestURL_UnwrapGoogleRedirect(t *testing.T) {
	raw := "https://www.google.com/url?q=https://example.com/path&utm_source=test&sa=U"
	got, err := URL(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://example.com/path"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestURL_DropTrackingParams(t *testing.T) {
	raw := "https://Example.com/page/?utm_source=x&b=2&a=1#a"
	got, err := URL(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://example.com/page?a=1&b=2"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

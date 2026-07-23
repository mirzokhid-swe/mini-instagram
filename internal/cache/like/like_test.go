package like

import "testing"

func TestParseStateKey(t *testing.T) {
	userID, postID, ok := ParseStateKey("like:12:34")
	if !ok || userID != 12 || postID != 34 {
		t.Fatalf("expected (12, 34, true), got (%d, %d, %v)", userID, postID, ok)
	}
}

func TestParseStateKey_Invalid(t *testing.T) {
	cases := []string{"like:12", "like:12:abc", "like-count:12:34", "junk"}
	for _, key := range cases {
		if _, _, ok := ParseStateKey(key); ok {
			t.Fatalf("expected key %q to be rejected", key)
		}
	}
}

func TestParseCountKey(t *testing.T) {
	postID, ok := ParseCountKey("like-count:42")
	if !ok || postID != 42 {
		t.Fatalf("expected (42, true), got (%d, %v)", postID, ok)
	}
}

func TestParseCountKey_Invalid(t *testing.T) {
	cases := []string{"like-count:abc", "like:12:34", "junk"}
	for _, key := range cases {
		if _, ok := ParseCountKey(key); ok {
			t.Fatalf("expected key %q to be rejected", key)
		}
	}
}

func TestKeyRoundTrip(t *testing.T) {
	if got := stateKey(12, 34); got != "like:12:34" {
		t.Fatalf("expected like:12:34, got %q", got)
	}
	if userID, postID, ok := ParseStateKey(stateKey(12, 34)); !ok || userID != 12 || postID != 34 {
		t.Fatalf("round trip failed: (%d, %d, %v)", userID, postID, ok)
	}

	if got := countKey(42); got != "like-count:42" {
		t.Fatalf("expected like-count:42, got %q", got)
	}
	if postID, ok := ParseCountKey(countKey(42)); !ok || postID != 42 {
		t.Fatalf("round trip failed: (%d, %v)", postID, ok)
	}
}

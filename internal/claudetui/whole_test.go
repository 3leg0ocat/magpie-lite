package claudetui

import (
	"errors"
	"strings"
	"testing"
	"unicode"
)

// TestWholeIn: the words Claude Code records of a pasted message are taken
// as whole whatever it trims, wraps, numbers or takes out, and a message
// that lost a block of its words is not (#5205, which lost 1 KB blocks),
// nor one that lost an image.
func TestWholeIn(t *testing.T) {
	long := strings.Repeat("line of words 中文 ", 400)
	cases := []struct {
		name   string
		sent   string
		said   []string
		whole  bool
		images int        // sent
		rows   []heardRow // said, with its images
	}{
		{"as sent", "hello there", []string{"hello there"}, true, 0, nil},
		{"trimmed and wrapped", "\n/command and more\n", []string{"/command and more"}, true, 0, nil},
		{"a long paste's tags", long, []string{"\n<pasted_content id=\"a1\">\n" + long + "\n</pasted_content id=\"a1\">\n"}, true, 0, nil},
		{"image marks", "look at this\n\n", []string{"[Image #1] look at this"}, true, 0, nil},
		{"a zero-width space taken out", "a\u200bb", []string{"ab"}, true, 0, nil},
		{"tabs as spaces", "a\tb", []string{"a    b"}, true, 0, nil},
		{"the user's mark", userMark + "go on", []string{"[user message]\ngo on"}, true, 0, nil},
		{"a later row is its", "the message", []string{"another row", "the message"}, true, 0, nil},
		{"no row read", "the message", nil, true, 0, nil},
		{"a block lost", long, []string{long[:1024] + long[2048:]}, false, 0, nil},
		{"its end lost", long, []string{long[:len(long)/2]}, false, 0, nil},
		{"taken as many pastes", long, []string{"<pasted_content id=\"a1\">\n" + long[:990] + "\n</pasted_content id=\"a1\">\n<pasted_content id=\"a2\">\n" + long[990:] + "\n</pasted_content id=\"a2\">"}, false, 0, nil},
		{"other words", "the message", []string{"another row"}, false, 0, nil},
		{"its images", "look\n", nil, true, 2, []heardRow{{"[Image #1] [Image #2] look", 2}}},
		{"an image lost", "look\n", nil, false, 2, []heardRow{{"[Image #1] look", 1}}},
		{"an image alone", "", nil, true, 1, []heardRow{{"[Image #1]", 1}}},
		{"a queued row says no images", "look\n", nil, true, 1, []heardRow{{"look", -1}}},
		{"images, but a word lost", "look here\n", nil, false, 1, []heardRow{{"look", 1}}},
		// image files' paths Claude Code took as the images (planner2, 2.1.319)
		{"paths taken as images", "C:\\a\\x.png\nlook\nC:\\a\\y.png C:\\a\\z.png\n", nil, true, 0, []heardRow{{"[Image #1] [Image #2] [Image #3]look", 3}}},
		{"paths taken beside images pasted", "look\n\nx.png\n", nil, true, 1, []heardRow{{"[Image #1] [Image #2]look", 2}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rows := c.rows
			for _, w := range c.said {
				rows = append(rows, heardRow{w, 0})
			}
			err := wholeIn(c.sent, c.images, rows)
			if (err == nil) != c.whole {
				t.Fatalf("wholeIn = %v, want whole %v", err, c.whole)
			}
			var cut *messageCut
			if err != nil && !errors.As(err, &cut) {
				t.Fatalf("wholeIn = %#v, want a messageCut", err)
			}
		})
	}
}

// BenchmarkWholeIn: a 256 KB message checked against its record, a long
// paste's, read in place.
func BenchmarkWholeIn(b *testing.B) {
	sent := strings.Repeat("line of words 中文 and more\n", 256<<10/30)
	said := []heardRow{{"\n<pasted_content id=\"a1\">\n" + sent + "</pasted_content id=\"a1\">\n", 0}}
	b.SetBytes(int64(len(sent)))
	b.ReportAllocs()
	for b.Loop() {
		if wholeIn(sent, 0, said) != nil {
			b.Fatal("not whole")
		}
	}
}

// TestWordsSkipRange: words reads U+3001 to U+FEFE as letters without
// asking unicode, as none of them is white space or a format character.
func TestWordsSkipRange(t *testing.T) {
	for r := rune(0x3001); r < 0xfeff; r++ {
		if unicode.IsSpace(r) || unicode.Is(unicode.Cf, r) {
			t.Fatalf("%U is white space or a format character", r)
		}
	}
}

// TestHold: what Claude Code answers while its message is checked is let
// out after the check when the message was whole, and never when it
// wasn't, the check's error alone told.
func TestHold(t *testing.T) {
	var b strings.Builder
	s := &shim{out: &b, sid: "s1"}
	s.hold()
	s.emit(map[string]any{"type": "stream_event"})
	if b.Len() > 0 {
		t.Fatal("an answer got out while its message was checked")
	}
	if s.unhold(nil) != nil || !strings.Contains(b.String(), "stream_event") {
		t.Fatalf("a whole message's answer wasn't let out: %q", b.String())
	}
	s.emit(map[string]any{"type": "assistant"})
	if !strings.Contains(b.String(), "assistant") {
		t.Fatal("held after the check")
	}
	b.Reset()
	s.hold()
	s.emit(map[string]any{"type": "stream_event"})
	err := s.unhold(&messageCut{got: 1, sent: 2})
	s.emit(map[string]any{"type": "result", "subtype": "success"})
	s.tell(map[string]any{"type": "result", "is_error": true, "result": Unpastable + err.Error()})
	if strings.Contains(b.String(), "stream_event") || strings.Contains(b.String(), "success") ||
		!strings.Contains(b.String(), Unpastable) {
		t.Fatalf("a cut message's answer got out: %q", b.String())
	}
}

package claudetui

import (
	"errors"
	"strings"
	"testing"
	"unicode"
)

// TestWholeIn: the words Claude Code records of a pasted message are taken
// as whole whatever it trims, wraps, numbers or takes out, and a message
// that lost a block of its words is not (#5205, which lost 1 KB blocks).
func TestWholeIn(t *testing.T) {
	long := strings.Repeat("line of words 中文 ", 400)
	cases := []struct {
		name  string
		sent  string
		said  []string
		whole bool
	}{
		{"as sent", "hello there", []string{"hello there"}, true},
		{"trimmed and wrapped", "\n/command and more\n", []string{"/command and more"}, true},
		{"a long paste's tags", long, []string{"\n<pasted_content id=\"a1\">\n" + long + "\n</pasted_content id=\"a1\">\n"}, true},
		{"image marks", "look at this\n\n", []string{"[Image #1] look at this"}, true},
		{"a zero-width space taken out", "a\u200bb", []string{"ab"}, true},
		{"tabs as spaces", "a\tb", []string{"a    b"}, true},
		{"the user's mark", userMark + "go on", []string{"[user message]\ngo on"}, true},
		{"a later row is its", "the message", []string{"another row", "the message"}, true},
		{"no row read", "the message", nil, true},
		{"a block lost", long, []string{long[:1024] + long[2048:]}, false},
		{"its end lost", long, []string{long[:len(long)/2]}, false},
		{"taken as many pastes", long, []string{"<pasted_content id=\"a1\">\n" + long[:990] + "\n</pasted_content id=\"a1\">\n<pasted_content id=\"a2\">\n" + long[990:] + "\n</pasted_content id=\"a2\">"}, false},
		{"other words", "the message", []string{"another row"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := wholeIn(c.sent, c.said)
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
	said := []string{"\n<pasted_content id=\"a1\">\n" + sent + "</pasted_content id=\"a1\">\n"}
	b.SetBytes(int64(len(sent)))
	b.ReportAllocs()
	for b.Loop() {
		if wholeIn(sent, said) != nil {
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

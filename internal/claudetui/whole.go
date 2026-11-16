package claudetui

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// A message is checked whole once Claude Code has it: the words its
// transcript holds of it against those pasted. A console that loses part
// of a paste (Windows 10's own ConPTY drops a long paste's marks and some
// of its blocks, #5205) gives Claude Code the rest, or the whole taken as
// many pastes with tags in the middle of its words, and the model answers
// that with nothing said. One that isn't whole ends the turn as
// Unpastable, before any of its reply, so the bridge asks it again as
// claude -p, which is sent the message itself.

// wholeWait is how long a message whose words no row holds yet waits for
// another row, in case the one read wasn't its.
const wholeWait = 2 * time.Second

// words reads a message's words one space apart, as Claude Code keeps what
// is pasted whatever it trims, wraps, numbers or takes out: a run of white
// space, a paste tag or an image mark is a space, none at the ends, and a
// format character (a zero-width space) is nothing. It reads in place, so
// two messages are compared without a copy of either.
type words struct {
	s     string
	i     int
	began bool // a letter was read
	space bool // a space comes before the next letter
}

// next is the next rune, ' ' for a space, -1 at the end.
func (w *words) next() rune {
	for w.i < len(w.s) {
		c := w.s[w.i]
		if c == '<' || c == '[' {
			if n := markAt(w.s[w.i:]); n > 0 {
				w.i += n
				w.space = w.began
				continue
			}
		}
		r, size := rune(c), 1
		if c >= utf8.RuneSelf {
			r, size = utf8.DecodeRuneInString(w.s[w.i:])
		}
		switch {
		case 0x3000 < r && r < 0xfeff:
			// CJK and the rest between: no white space, no format
			// character (unicode.IsSpace, unicode.Cf)
			if w.space {
				w.space = false
				return ' '
			}
		case c < utf8.RuneSelf && asciiSpace[c] || c >= utf8.RuneSelf && unicode.IsSpace(r):
			w.i += size
			w.space = w.began
			continue
		case c >= utf8.RuneSelf && unicode.Is(unicode.Cf, r):
			w.i += size
			continue
		case w.space:
			w.space = false
			return ' '
		}
		w.i += size
		w.began = true
		return r
	}
	return -1
}

// asciiSpace is unicode.IsSpace's white space below 0x80.
var asciiSpace = [utf8.RuneSelf]bool{'\t': true, '\n': true, '\v': true, '\f': true, '\r': true, ' ': true}

// markAt is the length of the paste tag (<pasted_content id="…">, or its
// end) or image mark ([Image #3]) s begins with, 0 for none.
func markAt(s string) int {
	if s == "" || s[0] != '<' && s[0] != '[' {
		return 0
	}
	var head, tail string
	var in func(byte) bool
	switch {
	case strings.HasPrefix(s, `<pasted_content id="`):
		head, tail, in = `<pasted_content id="`, `">`, isWord
	case strings.HasPrefix(s, `</pasted_content id="`):
		head, tail, in = `</pasted_content id="`, `">`, isWord
	case strings.HasPrefix(s, "[Image #"):
		head, tail, in = "[Image #", "]", isDigit
	default:
		return 0
	}
	n := len(head)
	for n < len(s) && in(s[n]) {
		n++
	}
	if n == len(head) || !strings.HasPrefix(s[n:], tail) {
		return 0
	}
	return n + len(tail)
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }
func isWord(c byte) bool  { return isDigit(c) || c == '_' || 'a' <= c|0x20 && c|0x20 <= 'z' }

// sameWords is whether a and b have the same words, read as words reads
// them; it stops at the first that differs.
func sameWords(a, b string) bool {
	x, y := words{s: a}, words{s: b}
	for {
		r := x.next()
		if r != y.next() {
			return false
		}
		if r < 0 {
			return true
		}
	}
}

// letters counts s's characters as words reads them, its spaces aside.
func letters(s string) int {
	w, n := words{s: s}, 0
	for r := w.next(); r >= 0; r = w.next() {
		if r != ' ' {
			n++
		}
	}
	return n
}

// messageCut is a message Claude Code recorded otherwise than it was
// pasted: got and sent count its characters, white space aside.
type messageCut struct{ got, sent int }

func (e *messageCut) Error() string {
	if e.got < e.sent {
		return fmt.Sprintf("Claude Code recorded %d of the message's %d characters%s", e.got, e.sent, cutHint())
	}
	return "Claude Code recorded the message otherwise than it was pasted" + cutHint()
}

// wholeIn is nil when one of the messages the transcript recorded since one
// was sent (said) has the words pasted (sent), or when it recorded none it
// could read.
func wholeIn(sent string, said []string) error {
	if len(said) == 0 {
		return nil
	}
	most := 0
	for _, w := range said {
		if sameWords(sent, w) {
			return nil
		}
		most = max(most, letters(w))
	}
	return &messageCut{got: most, sent: letters(sent)}
}

// checkWhole is wholeIn for a message just sent, given wholeWait for the
// row that is its. The rows' words are kept only while it waits (heard).
func (s *shim) checkWhole(sent string) error {
	defer func() {
		s.rowsMu.Lock()
		s.said = nil
		s.rowsMu.Unlock()
	}()
	by := time.After(wholeWait)
	for {
		s.rowsMu.Lock()
		said := slices.Clone(s.said)
		s.rowsMu.Unlock()
		err := wholeIn(sent, said)
		if err == nil {
			return nil
		}
		select {
		case <-s.rowsCh:
		case <-by:
			return err
		case <-s.exited:
			return err
		}
	}
}

// heard keeps the words of a message that reached the transcript while
// one sent is checked (said isn't nil from send to checkWhole's end); a
// row of one comes before its sawRow. words is read only then.
func (s *shim) heard(words func() string) {
	s.rowsMu.Lock()
	checking := s.said != nil
	s.rowsMu.Unlock()
	if !checking {
		return
	}
	w := words()
	if w == "" {
		return
	}
	s.rowsMu.Lock()
	if s.said != nil {
		s.said = append(s.said, w)
	}
	s.rowsMu.Unlock()
}

// heard tells an emitter that checks messages whole (shim) the words of
// one that reached the transcript.
func (t *translator) heard(words func() string) {
	if h, ok := t.out.(interface{ heard(func() string) }); ok {
		h.heard(words)
	}
}

// promptWords is the words of a user row's message: its text, or its
// text blocks one after another.
func promptWords(raw json.RawMessage) string {
	var m struct {
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	var s string
	if json.Unmarshal(m.Content, &s) == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	_ = json.Unmarshal(m.Content, &blocks)
	var b strings.Builder
	for _, bl := range blocks {
		if bl.Type == "text" {
			b.WriteString(bl.Text)
			b.WriteString("\n")
		}
	}
	return b.String()
}

package claudetui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
// pasted: got and sent count its characters, white space aside, and
// gotImages and sentImages its images when its words were whole.
type messageCut struct{ got, sent, gotImages, sentImages int }

func (e *messageCut) Error() string {
	switch {
	case e.gotImages < e.sentImages:
		return fmt.Sprintf("Claude Code recorded %d of the message's %d images%s", e.gotImages, e.sentImages, cutHint())
	case e.got < e.sent:
		return fmt.Sprintf("Claude Code recorded %d of the message's %d characters%s", e.got, e.sent, cutHint())
	}
	return "Claude Code recorded the message otherwise than it was pasted" + cutHint()
}

// heardRow is what the transcript recorded of a message: its words, and
// how many images it holds, -1 where a row doesn't say (a queued
// message's, which is its words alone).
type heardRow struct {
	words  string
	images int
}

// wholeIn is nil when one of the messages the transcript recorded since one
// was sent (said) has the words pasted (sent) and its images, or when it
// recorded none it could read, or one with more images than were pasted.
func wholeIn(sent string, images int, said []heardRow) error {
	if len(said) == 0 {
		return nil
	}
	got, gotImages, whole := 0, images, false
	for _, h := range said {
		// Claude Code took words of the paste as an image: a part of a
		// line that is an image file's path, by rules of its own that
		// change with its versions (2.1.319: a line split before each
		// absolute path in it), and recorded the image without them.
		// Its words aren't the paste's, and aren't compared.
		if h.images > images {
			return nil
		}
		if !sameWords(sent, h.words) {
			got = max(got, letters(h.words))
			continue
		}
		if h.images < 0 || h.images >= images {
			return nil
		}
		whole, gotImages = true, h.images
	}
	cut := &messageCut{got: got, sent: letters(sent), gotImages: gotImages, sentImages: images}
	if whole {
		cut.got = cut.sent
	}
	return cut
}

// checkWhole is wholeIn for a message just sent, given wholeWait for the
// row that is its.
func (s *shim) checkWhole(sent string, images int) error {
	s.hold()
	by := time.After(wholeWait)
	for {
		s.rowsMu.Lock()
		said := slices.Clone(s.said)
		s.rowsMu.Unlock()
		err := wholeIn(sent, images, said)
		if err == nil {
			return s.unhold(nil)
		}
		select {
		case <-s.rowsCh:
		case <-by:
			return s.unhold(err)
		case <-s.exited:
			return s.unhold(err)
		}
	}
}

// heard keeps what a message that reached the transcript holds while one
// sent is checked (said isn't nil from send to checkWhole's end); a row of
// one comes before its sawRow. row is read only then.
func (s *shim) heard(row func() heardRow) {
	s.rowsMu.Lock()
	checking := s.said != nil
	s.rowsMu.Unlock()
	if !checking {
		return
	}
	h := row()
	if h.words == "" && h.images <= 0 {
		return
	}
	s.rowsMu.Lock()
	if s.said != nil {
		s.said = append(s.said, h)
	}
	s.rowsMu.Unlock()
}

// heardAll ends what heard keeps, as send returns.
func (s *shim) heardAll() {
	s.rowsMu.Lock()
	s.said = nil
	s.rowsMu.Unlock()
}

// heard tells an emitter that checks messages whole (shim) what one that
// reached the transcript holds.
func (t *translator) heard(row func() heardRow) {
	if h, ok := t.out.(interface{ heard(func() heardRow) }); ok {
		h.heard(row)
	}
}

// promptRow is what a user row's message holds: its text, or its text
// blocks one after another, and its image blocks.
func promptRow(raw json.RawMessage) heardRow {
	var m struct {
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return heardRow{}
	}
	var s string
	if json.Unmarshal(m.Content, &s) == nil {
		return heardRow{words: s}
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	_ = json.Unmarshal(m.Content, &blocks)
	var h heardRow
	var b strings.Builder
	for _, bl := range blocks {
		switch bl.Type {
		case "text":
			b.WriteString(bl.Text)
			b.WriteString("\n")
		case "image":
			h.images++
		}
	}
	h.words = b.String()
	return h
}

// hold keeps what Claude Code answers while its message is checked: it
// asks Anthropic as soon as it records the message (25ms after), and a
// quick model answered words cut, its result let out, within the time
// waited for the message's other rows (wholeWait): the check's error came
// after the turn had ended (Haiku, 64K on Windows 10's console).
func (s *shim) hold() {
	s.outMu.Lock()
	s.held = new(bytes.Buffer)
	s.outMu.Unlock()
}

// unhold lets out what was held when the message was whole (err nil),
// and drops it, and all after it, when it wasn't: the turn's failure is
// told past it (tell).
func (s *shim) unhold(err error) error {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	if err != nil {
		s.held = io.Discard
		return err
	}
	if b, ok := s.held.(*bytes.Buffer); ok {
		_, _ = s.out.Write(b.Bytes())
	}
	s.held = nil
	return nil
}

// tell writes v past what is held (hold): a turn's failure.
func (s *shim) tell(v any) {
	b, _ := json.Marshal(v)
	s.outMu.Lock()
	defer s.outMu.Unlock()
	_, _ = s.out.Write(append(b, '\n'))
}

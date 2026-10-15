package mux

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// widePad marks the second column of a double-width character in the VT
// buffer. vt10x gives every rune one cell while the remote program (and the
// real terminal) give CJK characters two, so without it every Korean
// character would shift the rest of the line one column left.
const widePad = ''

var widePadBytes = []byte(string(widePad))

// cellWidth is the display width of r, matching how the remote side counts
// columns (East Asian ambiguous characters are narrow).
var cellWidth = func() func(rune) int {
	cond := runewidth.NewCondition()
	cond.EastAsianWidth = false
	return cond.RuneWidth
}()

// stringWidth is the display width of s by cellWidth.
func stringWidth(s string) int {
	w := 0
	for _, r := range s {
		w += cellWidth(r)
	}
	return w
}

type filterState int

const (
	stGround    filterState = iota
	stEsc                   // after ESC
	stEscInter              // ESC ( / ESC # …: one more byte
	stCSI                   // ESC [ … final byte
	stString                // OSC / DCS / APC / PM … BEL or ESC \
	stStringEsc             // ESC inside a string
)

// wideFilter inserts widePad after each double-width character of a PTY
// stream, leaving escape sequences untouched. It keeps state across chunks.
type wideFilter struct {
	state filterState
	pend  []byte // incomplete UTF-8 sequence from the previous chunk
}

func (f *wideFilter) apply(data []byte) []byte {
	out := make([]byte, 0, len(data)+len(data)/4)
	for i := 0; i < len(data); i++ {
		b := data[i]
		switch f.state {
		case stEsc:
			out = append(out, b)
			switch b {
			case '[':
				f.state = stCSI
			case ']', 'P', '_', '^', 'X':
				f.state = stString
			case '(', ')', '*', '+', '#', '%', ' ':
				f.state = stEscInter
			default:
				f.state = stGround
			}
			continue
		case stEscInter:
			out = append(out, b)
			f.state = stGround
			continue
		case stCSI:
			out = append(out, b)
			if b >= 0x40 && b <= 0x7e {
				f.state = stGround
			}
			continue
		case stString:
			out = append(out, b)
			if b == 0x07 {
				f.state = stGround
			} else if b == 0x1b {
				f.state = stStringEsc
			}
			continue
		case stStringEsc:
			out = append(out, b)
			f.state = stGround
			continue
		}

		// Ground state
		if len(f.pend) == 0 && b < utf8.RuneSelf {
			out = append(out, b)
			if b == 0x1b {
				f.state = stEsc
			}
			continue
		}
		f.pend = append(f.pend, b)
		if !utf8.FullRune(f.pend) {
			continue
		}
		r, size := utf8.DecodeRune(f.pend)
		out = append(out, f.pend[:size]...)
		rest := append([]byte(nil), f.pend[size:]...)
		f.pend = f.pend[:0]
		if r != utf8.RuneError && cellWidth(r) == 2 {
			out = append(out, widePadBytes...)
		}
		// An invalid byte decoded as RuneError(size 1): reprocess what followed it.
		if len(rest) > 0 {
			out = append(out, f.apply(rest)...)
		}
	}
	return out
}

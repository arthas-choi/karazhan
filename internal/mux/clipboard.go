package mux

import (
	"bytes"
	"encoding/base64"
)

// OSC 52 lets a program set the terminal's clipboard:
//
//	ESC ] 52 ; Pc ; Pd BEL      (or ESC \ instead of BEL)
//
// Pc names the selection (c, p, s, …) and Pd is the text in base64.
// vt10x has no clipboard, so clipFilter cuts these sequences out of a PTY
// stream and hands the decoded text to the host UI, which replays the
// sequence on the real terminal. All other output passes through unchanged.
//
// maxOSCLen bounds how much of an unterminated OSC is held back before it is
// flushed through as ordinary output.
const maxOSCLen = 1 << 20

type clipState int

const (
	csGround    clipState = iota
	csEsc                 // after ESC
	csEscInter            // ESC ( / ESC # …: one more byte
	csCSI                 // ESC [ … final byte
	csOSC                 // ESC ] … BEL or ESC \ (buffered in osc)
	csOSCEsc              // ESC inside an OSC
	csString              // DCS / APC / PM / SOS … passed through
	csStringEsc           // ESC inside such a string
)

// clipFilter removes OSC 52 sequences from a byte stream, keeping state
// across chunks.
type clipFilter struct {
	state clipState
	osc   []byte // OSC payload so far (without the ESC ] introducer)
}

// apply returns data without OSC 52 sequences, plus the decoded text of each
// complete OSC 52 sequence it removed, in order.
func (f *clipFilter) apply(data []byte) (out []byte, clips []string) {
	out = make([]byte, 0, len(data))
	for _, b := range data {
		switch f.state {
		case csGround:
			if b == 0x1b {
				// Held back (possibly across chunks) until the next byte
				// says whether an OSC follows.
				f.state = csEsc
				continue
			}
			out = append(out, b)
		case csEsc:
			switch b {
			case ']':
				f.osc = f.osc[:0]
				f.state = csOSC
				continue
			case '[':
				f.state = csCSI
			case 'P', '_', '^', 'X':
				f.state = csString
			case '(', ')', '*', '+', '#', '%', ' ':
				f.state = csEscInter
			default:
				f.state = csGround
			}
			out = append(out, 0x1b, b)
		case csEscInter:
			out = append(out, b)
			f.state = csGround
		case csCSI:
			out = append(out, b)
			if b >= 0x40 && b <= 0x7e {
				f.state = csGround
			}
		case csString:
			out = append(out, b)
			if b == 0x07 {
				f.state = csGround
			} else if b == 0x1b {
				f.state = csStringEsc
			}
		case csStringEsc:
			out = append(out, b)
			f.state = csGround

		case csOSC:
			switch {
			case b == 0x07:
				out, clips = f.finishOSC(out, clips, []byte{0x07})
			case b == 0x1b:
				f.state = csOSCEsc
			default:
				f.osc = append(f.osc, b)
				if len(f.osc) > maxOSCLen {
					// Give up on this one: emit it as-is and treat the rest
					// as a plain string until it ends.
					out = append(out, 0x1b, ']')
					out = append(out, f.osc...)
					f.osc = f.osc[:0]
					f.state = csString
				}
			}
		case csOSCEsc:
			if b == '\\' {
				out, clips = f.finishOSC(out, clips, []byte{0x1b, '\\'})
			} else {
				// Not a terminator; keep both bytes in the payload.
				f.osc = append(f.osc, 0x1b, b)
				f.state = csOSC
			}
		}
	}
	return out, clips
}

// finishOSC handles a completed OSC: a clipboard write is recorded and
// dropped from the stream, anything else is emitted unchanged.
func (f *clipFilter) finishOSC(out []byte, clips []string, term []byte) ([]byte, []string) {
	f.state = csGround
	if text, ok := parseOSC52(f.osc); ok {
		if text != "" {
			clips = append(clips, text)
		}
		return out, clips
	}
	out = append(out, 0x1b, ']')
	out = append(out, f.osc...)
	out = append(out, term...)
	return out, clips
}

// parseOSC52 reports whether payload is an OSC 52 clipboard sequence and,
// for a clipboard write, the decoded text. Queries ("?") and undecodable
// payloads are swallowed with an empty text.
func parseOSC52(payload []byte) (string, bool) {
	if !bytes.HasPrefix(payload, []byte("52;")) {
		return "", false
	}
	rest := payload[3:]
	i := bytes.IndexByte(rest, ';')
	if i < 0 {
		return "", true
	}
	data := rest[i+1:]
	if len(data) == 0 || bytes.Equal(data, []byte("?")) {
		return "", true
	}
	dec, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		dec, err = base64.RawStdEncoding.DecodeString(string(data))
	}
	if err != nil {
		return "", true
	}
	return string(dec), true
}

// OSC52 returns the escape sequence that puts text on the terminal's clipboard.
func OSC52(text string) string {
	return "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
}

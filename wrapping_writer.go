package command

import (
	"fmt"
	"strings"
	"unicode"
)

type WrappingWriter struct {
	data                   []rune
	width                  int
	remainingToNextNewLine int
	linePrefix             string
}

func NewWrappingWriter(width int) (*WrappingWriter, error) {
	if width <= 0 {
		return nil, fmt.Errorf("illegal width: %d", width)
	}
	return &WrappingWriter{data: nil, width: width, remainingToNextNewLine: width}, nil
}

func (w *WrappingWriter) SetLinePrefix(prefix string) error {
	if len(prefix) >= w.width {
		return fmt.Errorf("invalid prefix '%s': too larger for width %d", prefix, w.width)
	} else if strings.Contains(prefix, "\n") {
		return fmt.Errorf("invalid prefix '%s': cannot contain new lines", prefix)
	}
	w.linePrefix = prefix
	return nil
}

func (w *WrappingWriter) Write(p []byte) (n int, err error) {
	srcRunes := []rune(string(p))
	for i := 0; i < len(srcRunes); i++ {
		r := srcRunes[i]
		if r == '\n' {
			if len(w.data) == 0 || (i > 0 && w.data[len(w.data)-1] == '\n') {
				w.data = append(w.data, []rune(w.linePrefix)...)
			}
			w.data = append(w.data, r)
			w.remainingToNextNewLine = w.width
		} else if w.remainingToNextNewLine == 0 {
			for j := len(w.data) - 1; j >= 0; j-- {
				rr := w.data[j]
				if rr == '\n' {
					// Current line has no space; just keep writing this line without splitting it
					w.data = append(w.data, r)
					break
				} else if len(w.data)-j+len(w.linePrefix) >= w.width {
					// current line is already at width-length (including prefix) - just keep writing
					w.data = append(w.data, r)
					break
				} else if unicode.IsSpace(rr) {
					var runesBeforeSpace, runesAfterSpace []rune
					runesBeforeSpace = w.data[0 : j+1]
					if j < len(w.data)-1 {
						runesAfterSpace = w.data[j+1:]
					}
					w.data = make([]rune, 0, len(runesBeforeSpace)+len(runesAfterSpace)+1)
					w.data = append(w.data, runesBeforeSpace...)
					w.data = append(w.data, '\n')
					w.data = append(w.data, []rune(w.linePrefix)...)
					w.data = append(w.data, runesAfterSpace...)
					w.data = append(w.data, r)

					// Remaining characters now equal width minus text after last space, minus the char we just wrote
					w.remainingToNextNewLine = w.width - len(w.linePrefix) - len(runesAfterSpace) - 1
					if w.remainingToNextNewLine < 0 {
						w.remainingToNextNewLine = 0
					}
					break
				}
			}
		} else {
			if len(w.data) == 0 || w.data[len(w.data)-1] == '\n' {
				w.data = append(w.data, []rune(w.linePrefix)...)
				w.remainingToNextNewLine -= len(w.linePrefix)
			}
			w.data = append(w.data, r)
			w.remainingToNextNewLine--
		}
	}
	return len(p), nil
}

func (w *WrappingWriter) String() string {
	return string(w.data)
}

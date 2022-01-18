package lexer

import (
	"log"
	"strconv"
)

const (
	TYPE_INT  = 0
	TYPE_PLUS = 1
)

var (
	input string
	pos   int
)

func getCh() (ch rune, end bool) {
	if pos == len(input) {
		return ch, true
	}
	pos++
	ch = []rune(input)[pos-1]
	return ch, false
}

func getChSkipEmpty() (ch rune, end bool) {
	ch, end = getCh()
	if end {
		return
	}
	if ch == ' ' {
		return getChSkipEmpty()
	}
	return
}
func isLetter(ch rune) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}
func isNum(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func retract() {
	pos--
}

func Scan() (code int, val interface{}, eos bool) {
	ch, end := getChSkipEmpty()
	if end {
		eos = end
		return
	}
	if isLetter(ch) {
		log.Fatalf("unexpected letter %c in pos %d", ch, pos)
	}
	if isNum(ch) {
		i := []rune{ch}
		for {
			c, end := getCh()
			if end {
				break
			}
			if !isNum(c) {
				retract()
				break
			}
			i = append(i, c)
		}
		interger, _ := strconv.Atoi(string(i))
		return TYPE_INT, interger, end
	}
	switch ch {
	case '+':
		return TYPE_PLUS, nil, end
	default:
		log.Fatalf("unrecognized letter %c in pos %d", ch, pos)
	}
	return

}

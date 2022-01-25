package lexer

import (
	"fmt"
	"log"
)

const (
	TYPE_INT  = 0
	TYPE_PLUS = 1
	TYPE_SUB  = 2
	TYPE_MUL  = 3
	TYPE_DIV  = 4
	TYPE_LP   = 5
	TYPE_RP   = 6
)

var (
	input   string
	pos     int
	ErrEOS  = fmt.Errorf("eos error")
	ErrTYPE = fmt.Errorf("the next token doesn't match the expected type")
)

func SetInput(s string) {
	pos = 0
	input = s
}

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

func Retract(i int) {
	pos -= i
}

func ScanType(code int) (token string, err error) {
	c, t, e := Scan()
	if c == code {
		return t, nil
	} else if e {
		return "", ErrEOS
	}
	pos -= len(t)
	return "", ErrTYPE
}

func Scan() (code int, token string, eos bool) {
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
				pos--
				break
			}
			i = append(i, c)
		}
		return TYPE_INT, string(i), end
	}
	switch ch {
	case '+':
		return TYPE_PLUS, "+", end
	case '-':
		return TYPE_SUB, "-", end
	case '*':
		return TYPE_MUL, "*", end
	case '/':
		return TYPE_DIV, "/", end
	case '(':
		return TYPE_LP, "(", end
	case ')':
		return TYPE_RP, ")", end
	default:
		log.Fatalf("unrecognized letter %c in pos %d", ch, pos)
	}
	return

}

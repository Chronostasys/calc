package lexer

import (
	"fmt"
	"log"
)

const (
	TYPE_INT       = 0
	TYPE_PLUS      = 1
	TYPE_SUB       = 2
	TYPE_MUL       = 3
	TYPE_DIV       = 4
	TYPE_LP        = 5  // "("
	TYPE_RP        = 6  // ")"
	TYPE_ASSIGN    = 7  // "="
	TYPE_RES_VAR   = 8  // "var"
	TYPE_RES_INT   = 9  // "int"
	TYPE_NL        = 10 // "\n"
	TYPE_VAR       = 11
	TYPE_FLOAT     = 12
	TYPE_RES_FLOAT = 13
	TYPE_RES_FUNC  = 14 // "func"
	TYPE_LB        = 15 // "{"
	TYPE_RB        = 16 // "}"
	TYPE_COMMA     = 17 // ","
	TYPE_RES_RET   = 18 // "return"
	TYPE_RES_VOID  = 19 // "void"
	TYPE_RES_TRUE  = 20 // "true"
	TYPE_RES_FALSE = 21 // "false"
	TYPE_AND       = 22 // "&&"
	TYPE_OR        = 23 // "||"
	TYPE_EQ        = 24 // "=="
	TYPE_RES_BOOL  = 25 // "bool"
)

var (
	input    string
	pos      int
	reserved = map[string]int{
		"var":    TYPE_RES_VAR,
		"int":    TYPE_RES_INT,
		"float":  TYPE_RES_FLOAT,
		"func":   TYPE_RES_FUNC,
		"return": TYPE_RES_RET,
		"void":   TYPE_RES_VOID,
		"true":   TYPE_RES_TRUE,
		"false":  TYPE_RES_FALSE,
		"bool":   TYPE_RES_BOOL,
	}
	reservedTypes = map[string]int{
		"int":   TYPE_RES_INT,
		"float": TYPE_RES_FLOAT,
		"void":  TYPE_RES_VOID,
		"bool":  TYPE_RES_BOOL,
	}
	ErrEOS  = fmt.Errorf("eos error")
	ErrTYPE = fmt.Errorf("the next token doesn't match the expected type")
)

func IsResType(token string) (code int, ok bool) {
	code, ok = reservedTypes[token]
	return
}

func SetInput(s string) {
	pos = 0
	input = s
}

func Peek() (ch rune, end bool) {
	if pos >= len(input) {
		return ch, true
	}
	ch = []rune(input)[pos]
	return ch, false
}

func getCh() (ch rune, end bool) {
	defer func() {
		pos++
	}()
	return Peek()
}

func getChSkipEmpty() (ch rune, end bool) {
	ch, end = getCh()
	if end {
		return
	}
	if ch == ' ' || ch == '\t' {
		return getChSkipEmpty()
	}
	return
}
func isLetter(ch rune) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}
func isLetterOrUnderscore(ch rune) bool {
	return isLetter(ch) || ch == '_'
}
func isNum(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func Retract(i int) {
	pos -= i
}

type Checkpoint struct {
	pos int
}

func SetCheckpoint() Checkpoint {
	return Checkpoint{
		pos: pos,
	}
}
func GobackTo(c Checkpoint) {
	pos = c.pos
}

func ScanType(code int) (token string, err error) {
	ch := SetCheckpoint()
	c, t, e := Scan()
	if c == code {
		return t, nil
	} else if e {
		return "", ErrEOS
	}
	// fmt.Println(pos, t)
	GobackTo(ch)
	return "", ErrTYPE
}

func PrintPos() {
	println(pos)
}

func Scan() (code int, token string, eos bool) {
	ch, end := getChSkipEmpty()
	if end {
		eos = end
		return
	}
	if isLetterOrUnderscore(ch) {
		i := []rune{ch}
		for {
			c, end := getCh()
			if end {
				break
			}
			if !isLetterOrUnderscore(c) && !isNum(c) {
				pos--
				break
			}
			i = append(i, c)
		}
		token = string(i)
		if tp, ok := reserved[token]; ok {
			return tp, token, end
		}
		return TYPE_VAR, string(i), end
	}
	if isNum(ch) {
		i := []rune{ch}
		t := TYPE_INT
		for {
			c, end := getCh()
			if end {
				break
			}
			if c == '.' {
				i = append(i, c)
				t = TYPE_FLOAT
				continue
			}
			if !isNum(c) {
				pos--
				break
			}
			i = append(i, c)
		}
		return t, string(i), end
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
	case '=':
		if ne, _ := Peek(); ne == '=' {
			getCh()
			return TYPE_EQ, "==", end
		}
		return TYPE_ASSIGN, "=", end
	case '\n':
		return TYPE_NL, "\n", end
	case '\r':
		c, e := Peek()
		if !e && c == '\n' {
			pos++
			return TYPE_NL, "\n", e
		}
	case '{':
		return TYPE_LB, "{", end
	case '}':
		return TYPE_RB, "}", end
	case ',':
		return TYPE_COMMA, ",", end
	case '&':
		if ne, _ := Peek(); ne == '&' {
			getCh()
			return TYPE_AND, "&&", end
		}
	case '|':
		if ne, _ := Peek(); ne == '|' {
			getCh()
			return TYPE_OR, "||", end
		}
	}
	log.Fatalf("unrecognized letter %c in pos %d", ch, pos)
	return

}

package lexer

import (
	"fmt"
	"log"
)

const (
	TYPE_INT = iota
	TYPE_PLUS
	TYPE_SUB
	TYPE_MUL
	TYPE_DIV
	TYPE_LP      // "("
	TYPE_RP      // ")"
	TYPE_ASSIGN  // "="
	TYPE_RES_VAR // "var"
	TYPE_RES_INT // "int"
	TYPE_NL      // "\n"
	TYPE_VAR
	TYPE_FLOAT
	TYPE_RES_FLOAT
	TYPE_RES_FUNC   // "func"
	TYPE_LB         // "{"
	TYPE_RB         // "}"
	TYPE_COMMA      // ","
	TYPE_RES_RET    // "return"
	TYPE_RES_VOID   // "void"
	TYPE_RES_TRUE   // "true"
	TYPE_RES_FALSE  // "false"
	TYPE_AND        // "&&"
	TYPE_OR         // "||"
	TYPE_EQ         // "=="
	TYPE_RES_BOOL   // "bool"
	TYPE_LG         // ">"
	TYPE_SM         // "<"
	TYPE_LEQ        // ">="
	TYPE_SEQ        // "<="
	TYPE_NOT        // "!"
	TYPE_NEQ        // "!="
	TYPE_RES_IF     // "if"
	TYPE_RES_EL     // "else"
	TYPE_DEAS       // ":="
	TYPE_RES_FOR    // "for"
	TYPE_SEMI       // ";"
	TYPE_RES_BR     // "break"
	TYPE_RES_CO     // "continue"
	TYPE_RES_TYPE   // "type"
	TYPE_RES_STRUCT // "struct"
)

var (
	input    string
	pos      int
	reserved = map[string]int{
		"var":      TYPE_RES_VAR,
		"int":      TYPE_RES_INT,
		"float":    TYPE_RES_FLOAT,
		"func":     TYPE_RES_FUNC,
		"return":   TYPE_RES_RET,
		"void":     TYPE_RES_VOID,
		"true":     TYPE_RES_TRUE,
		"false":    TYPE_RES_FALSE,
		"bool":     TYPE_RES_BOOL,
		"if":       TYPE_RES_IF,
		"else":     TYPE_RES_EL,
		"for":      TYPE_RES_FOR,
		"break":    TYPE_RES_BR,
		"continue": TYPE_RES_CO,
		"type":     TYPE_RES_TYPE,
		"struct":   TYPE_RES_STRUCT,
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

func PeekToken() (code int, token string, eos bool) {
	ch := SetCheckpoint()
	defer func() {
		GobackTo(ch)
	}()
	return Scan()
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
			if c == '.' {
				i = append(i, c)
				continue
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
	case '>':
		if ne, _ := Peek(); ne == '=' {
			getCh()
			return TYPE_LEQ, ">=", end
		}
		return TYPE_LG, ">", end
	case '<':
		if ne, _ := Peek(); ne == '=' {
			getCh()
			return TYPE_SEQ, "<=", end
		}
		return TYPE_SM, "<", end
	case '!':
		if ne, _ := Peek(); ne == '=' {
			getCh()
			return TYPE_NEQ, "!=", end
		}
		return TYPE_NOT, "!", end
	case ':':
		if ne, _ := Peek(); ne == '=' {
			getCh()
			return TYPE_DEAS, ":=", end
		}
	case ';':
		return TYPE_SEMI, ";", end
	}
	log.Fatalf("unrecognized letter %c in pos %d", ch, pos)
	return

}

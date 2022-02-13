package lexer

import (
	"fmt"
	"log"
)

const (
	TYPE_INT           = iota
	TYPE_PLUS          // "+"
	TYPE_SUB           // "-"
	TYPE_MUL           // "*"
	TYPE_DIV           // "/"
	TYPE_LP            // "("
	TYPE_RP            // ")"
	TYPE_ASSIGN        // "="
	TYPE_RES_VAR       // "var"
	TYPE_RES_INT       // "int"
	TYPE_NL            // "\n"
	TYPE_VAR           // "([a-z]|[A-Z])([a-z]|[A-Z]|[0-9])*"
	TYPE_FLOAT         // 小数，x.y这种
	TYPE_RES_FLOAT     // "float"
	TYPE_RES_FUNC      // "func"
	TYPE_LB            // "{"
	TYPE_RB            // "}"
	TYPE_COMMA         // ","
	TYPE_RES_RET       // "return"
	TYPE_RES_VOID      // "void"
	TYPE_RES_TRUE      // "true"
	TYPE_RES_FALSE     // "false"
	TYPE_AND           // "&&"
	TYPE_OR            // "||"
	TYPE_EQ            // "=="
	TYPE_RES_BOOL      // "bool"
	TYPE_LG            // ">"
	TYPE_SM            // "<"
	TYPE_LEQ           // ">="
	TYPE_SEQ           // "<="
	TYPE_NOT           // "!"
	TYPE_NEQ           // "!="
	TYPE_RES_IF        // "if"
	TYPE_RES_EL        // "else"
	TYPE_DEAS          // ":="
	TYPE_RES_FOR       // "for"
	TYPE_SEMI          // ";"
	TYPE_RES_BR        // "break"
	TYPE_RES_CO        // "continue"
	TYPE_RES_TYPE      // "type"
	TYPE_RES_STRUCT    // "struct"
	TYPE_COLON         // ":"
	TYPE_LSB           // "["
	TYPE_RSB           // "]"
	TYPE_ESP           // "&"
	TYPE_RES_INT32     // "int32"
	TYPE_RES_FLOAT32   // "float32"
	TYPE_RES_INT64     // "int64"
	TYPE_RES_FLOAT64   // "float64"
	TYPE_RES_BYTE      // "byte"
	TYPE_DOT           // "."
	TYPE_RES_THIS      // "this"
	TYPE_RES_INTERFACE // "interface"
	TYPE_RES_NIL       // "nil"
	TYPE_RES_PKG       // "pkg"
	TYPE_STR           // ant quoted string
	TYPE_RES_STR       // "string"
	TYPE_RES_IMPORT    // "import"
	TYPE_RES_OP        // "op"
)

var (
	reserved = map[string]int{
		"var":       TYPE_RES_VAR,
		"int":       TYPE_RES_INT,
		"float":     TYPE_RES_FLOAT,
		"func":      TYPE_RES_FUNC,
		"return":    TYPE_RES_RET,
		"void":      TYPE_RES_VOID,
		"true":      TYPE_RES_TRUE,
		"false":     TYPE_RES_FALSE,
		"bool":      TYPE_RES_BOOL,
		"if":        TYPE_RES_IF,
		"else":      TYPE_RES_EL,
		"for":       TYPE_RES_FOR,
		"break":     TYPE_RES_BR,
		"continue":  TYPE_RES_CO,
		"type":      TYPE_RES_TYPE,
		"struct":    TYPE_RES_STRUCT,
		"int32":     TYPE_RES_INT32,
		"int64":     TYPE_RES_INT64,
		"float32":   TYPE_RES_FLOAT32,
		"float64":   TYPE_RES_FLOAT64,
		"byte":      TYPE_RES_BYTE,
		"this":      TYPE_RES_THIS,
		"interface": TYPE_RES_INTERFACE,
		"nil":       TYPE_RES_NIL,
		"package":   TYPE_RES_PKG,
		"string":    TYPE_RES_STR,
		"import":    TYPE_RES_IMPORT,
		"op":        TYPE_RES_OP,
	}
	reservedTypes = map[string]int{
		"int":     TYPE_RES_INT,
		"float":   TYPE_RES_FLOAT,
		"void":    TYPE_RES_VOID,
		"bool":    TYPE_RES_BOOL,
		"int32":   TYPE_RES_INT32,
		"int64":   TYPE_RES_INT64,
		"float32": TYPE_RES_FLOAT32,
		"float64": TYPE_RES_FLOAT64,
		"byte":    TYPE_RES_BYTE,
		"string":  TYPE_RES_STR,
	}
	ErrEOS  = fmt.Errorf("eos error")
	ErrTYPE = fmt.Errorf("the next token doesn't match the expected type")
)

type Lexer struct {
	input string
	pos   int
}

func IsResType(token string) (code int, ok bool) {
	code, ok = reservedTypes[token]
	return
}

func (l *Lexer) SetInput(s string) {
	l.pos = 0
	l.input = s
}

func (l *Lexer) Peek() (ch rune, end bool) {
	if l.pos >= len(l.input) {
		return ch, true
	}
	ch = []rune(l.input)[l.pos]
	return ch, false
}

func (l *Lexer) PeekToken() (code int, token string, eos bool) {
	ch := l.SetCheckpoint()
	defer func() {
		l.GobackTo(ch)
	}()
	return l.Scan()
}

func (l *Lexer) getCh() (ch rune, end bool) {
	defer func() {
		l.pos++
	}()
	return l.Peek()
}

func (l *Lexer) getChSkipEmpty() (ch rune, end bool) {
	ch, end = l.getCh()
	if end {
		return
	}
	if ch == ' ' || ch == '\t' {
		return l.getChSkipEmpty()
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

type Checkpoint struct {
	pos int
}

func (l *Lexer) SetCheckpoint() Checkpoint {
	return Checkpoint{
		pos: l.pos,
	}
}
func (l *Lexer) GobackTo(c Checkpoint) {
	l.pos = c.pos
}

func (l *Lexer) ScanType(code int) (token string, err error) {
	ch := l.SetCheckpoint()
	c, t, e := l.Scan()
	if c == code {
		return t, nil
	} else if e {
		return "", ErrEOS
	}
	// fmt.Println(pos, t)
	l.GobackTo(ch)
	return "", ErrTYPE
}

func (l *Lexer) PrintCurrent() {
	start, end := l.pos-10, l.pos+10
	if start < 0 {
		start = 0
	}
	if end > len(l.input) {
		end = len(l.input)
	}
	fmt.Println(l.input[start:end])
}

func (l *Lexer) Scan() (code int, token string, eos bool) {
	ch, end := l.getChSkipEmpty()
	if end {
		eos = end
		return
	}
	if ch == '"' {
		i := []rune{}
		for {
			c, end := l.getCh()
			if end {
				break
			}
			if c == '\\' {
				c, end := l.getCh()
				if end {
					break
				}
				switch c {
				case '"', '\'':
					i = append(i, c)
				case 't':
					i = append(i, '\t')
				case 'n':
					i = append(i, '\n')
				case 'r':
					i = append(i, '\r')
				case '\\':
					i = append(i, '\\')
				case '0':
					i = append(i, '\x00')
				default:
					panic(fmt.Sprintf("unknown escape symbol %c", c))
				}
				continue

			}
			if c == '"' {
				break
			}
			i = append(i, c)
		}
		return TYPE_STR, string(i), end
	}
	if isLetterOrUnderscore(ch) {
		i := []rune{ch}
		for {
			c, end := l.getCh()
			if end {
				break
			}
			if !isLetterOrUnderscore(c) && !isNum(c) {
				l.pos--
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
			c, end := l.getCh()
			if end {
				break
			}
			if c == '.' {
				i = append(i, c)
				t = TYPE_FLOAT
				continue
			}
			if !isNum(c) {
				l.pos--
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
		if ne, _ := l.Peek(); ne == '=' {
			l.getCh()
			return TYPE_EQ, "==", end
		}
		return TYPE_ASSIGN, "=", end
	case '\n':
		return TYPE_NL, "\n", end
	case '\r':
		c, e := l.Peek()
		if !e && c == '\n' {
			l.pos++
			return TYPE_NL, "\n", e
		}
	case '{':
		return TYPE_LB, "{", end
	case '}':
		return TYPE_RB, "}", end
	case ',':
		return TYPE_COMMA, ",", end
	case '&':
		if ne, _ := l.Peek(); ne == '&' {
			l.getCh()
			return TYPE_AND, "&&", end
		}
		return TYPE_ESP, "&", end
	case '|':
		if ne, _ := l.Peek(); ne == '|' {
			l.getCh()
			return TYPE_OR, "||", end
		}
	case '>':
		if ne, _ := l.Peek(); ne == '=' {
			l.getCh()
			return TYPE_LEQ, ">=", end
		}
		return TYPE_LG, ">", end
	case '<':
		if ne, _ := l.Peek(); ne == '=' {
			l.getCh()
			return TYPE_SEQ, "<=", end
		}
		return TYPE_SM, "<", end
	case '!':
		if ne, _ := l.Peek(); ne == '=' {
			l.getCh()
			return TYPE_NEQ, "!=", end
		}
		return TYPE_NOT, "!", end
	case ':':
		if ne, _ := l.Peek(); ne == '=' {
			l.getCh()
			return TYPE_DEAS, ":=", end
		}
		return TYPE_COLON, ":", end
	case ';':
		return TYPE_SEMI, ";", end
	case '[':
		return TYPE_LSB, "[", end
	case ']':
		return TYPE_RSB, "]", end
	case '.':
		return TYPE_DOT, ".", end
	}
	log.Fatalf("unrecognized letter %c inl.pos %d", ch, l.pos)
	return

}
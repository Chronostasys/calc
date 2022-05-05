package ast

type syntaxErr struct {
	ErrBlockNode
}

func (e *syntaxErr) Error() string {
	return e.Message
}

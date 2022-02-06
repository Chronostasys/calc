package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronostasys/calculator_go/ast"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"

	"github.com/Chronostasys/calculator_go/lexer"
)

func ParseInt(s string) (int64, *types.IntType, error) {
	bw := 8
	for {
		re, err := strconv.ParseInt(s, 10, bw)
		if err == nil {
			return re, types.NewInt(uint64(bw)), err
		} else {
			if bw == 64 {
				return 0, nil, err
			}
			bw *= 2
		}
	}
}

func number() (n ast.Node) {
	ch := lexer.SetCheckpoint()
	n, err := runWithCatch2(takeValExp)
	if err == nil {
		return n
	}
	code, t1, eos := lexer.Scan()
	if eos {
		panic("eos")
	}
	switch code {
	case lexer.TYPE_FLOAT:
		i, err := strconv.ParseFloat(t1, 32)
		tp := types.Float
		if err != nil {
			i, err = strconv.ParseFloat(t1, 64)
			if err != nil {
				panic(err)
			}
			tp = types.Double
		}
		return &ast.NumNode{Val: constant.NewFloat(tp, i)}
	case lexer.TYPE_INT:
		i, tp, err := ParseInt(t1)
		if err != nil {
			panic(err)
		}
		return &ast.NumNode{Val: constant.NewInt(tp, i)}

	}
	lexer.GobackTo(ch)
	_, err = lexer.ScanType(lexer.TYPE_LP)
	if err != nil {
		panic(err)
	}
	i := exp()
	_, err = lexer.ScanType(lexer.TYPE_RP)
	if err != nil {
		panic(err)
	}
	return i
}

func factor() ast.Node {
	a := symbol()
	ch := lexer.SetCheckpoint()
	code, _, eos := lexer.Scan()
	for !eos && code == lexer.TYPE_DIV || code == lexer.TYPE_MUL {
		b := symbol()
		a = &ast.BinNode{
			Op:    code,
			Left:  a,
			Right: b,
		}
		ch = lexer.SetCheckpoint()
		code, _, eos = lexer.Scan()
	}
	if !eos {
		lexer.GobackTo(ch)
	}
	return a
}

func exp() ast.Node {
	a := factor()
	ch := lexer.SetCheckpoint()
	code, _, eos := lexer.Scan()
	for !eos && code == lexer.TYPE_PLUS || code == lexer.TYPE_SUB {
		b := factor()
		a = &ast.BinNode{
			Op:    code,
			Left:  a,
			Right: b,
		}
		ch = lexer.SetCheckpoint()
		code, _, eos = lexer.Scan()
	}
	if !eos {
		lexer.GobackTo(ch)
	}
	return a
}

func symbol() ast.Node {
	ch := lexer.SetCheckpoint()
	code, _, eos := lexer.Scan()
	if eos {
		panic(lexer.ErrEOS)
	}
	if code == lexer.TYPE_PLUS || code == lexer.TYPE_SUB {
		return &ast.UnaryNode{Op: code, Child: number()}
	}
	lexer.GobackTo(ch)
	return number()
}

func assign() (n ast.Node, err error) {
	c := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(c)
		}
	}()
	level := 0
	for {
		_, err = lexer.ScanType(lexer.TYPE_MUL)
		if err != nil {
			break
		}
		level++
	}
	node, err := runWithCatch2(varChain)
	if err != nil {
		return nil, err
	}
	_, err = lexer.ScanType(lexer.TYPE_ASSIGN)
	if err != nil {
		return nil, err
	}
	r := allexp()
	return &ast.BinNode{
		Left:  &ast.TakeValNode{Node: node, Level: level},
		Op:    lexer.TYPE_ASSIGN,
		Right: r,
	}, nil
}

func empty() ast.Node {
	_, err := lexer.ScanType(lexer.TYPE_NL)
	if err != nil {
		panic(err)
	}
	return &ast.EmptyNode{}
}

func define() (n ast.Node, err error) {
	c := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(c)
		}
		if err == nil {
			empty()
		}
	}()
	_, err = lexer.ScanType(lexer.TYPE_RES_VAR)
	if err != nil {
		return nil, err
	}
	id, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	tp, err := allTypes()
	if err != nil {
		panic(err)
	}
	return &ast.DefineNode{ID: id, TP: tp}, nil
}

func statement() ast.Node {
	ast, err := continueST()
	if err == nil {
		return ast
	}
	ast, err = breakST()
	if err == nil {
		return ast
	}
	ast, err = forloop()
	if err == nil {
		return ast
	}
	ast, err = defineAndAssign()
	if err == nil {
		return ast
	}
	ast, err = ifstatement()
	if err == nil {
		return ast
	}
	ast, err = assign()
	if err == nil {
		return ast
	}
	ast, err = define()
	if err == nil {
		return ast
	}
	ast, err = returnST()
	if err == nil {
		return ast
	}
	ch := lexer.SetCheckpoint()
	c, t, _ := lexer.Scan()
	if c == lexer.TYPE_VAR {
		lexer.GobackTo(ch)
		cf := callFunc()
		empty()
		return cf
	} else if c == lexer.TYPE_NL {
		lexer.GobackTo(ch)
		return empty()
	}
	panic(fmt.Sprintf("parse fail %s", t))
}

func statementList() ast.Node {
	n := &ast.SLNode{}
	n.Children = append(n.Children, statement())
	ch := lexer.SetCheckpoint()
	c, _, _ := lexer.Scan()
	lexer.GobackTo(ch)
	if c == lexer.TYPE_RB {
		return n
	}
	n.Children = append(n.Children, statementList())
	return n
}

func program() *ast.ProgramNode {
	n := &ast.ProgramNode{}
	//
	for {
		ch := lexer.SetCheckpoint()
		c, _, eos := lexer.Scan()
		if c == lexer.TYPE_NL {
			continue
		}
		lexer.GobackTo(ch)
		if eos {
			break
		}
		ast, err := structDef()
		if err == nil {
			n.Children = append(n.Children, ast)
		} else {
			n.Children = append(n.Children, function())
		}
	}
	return n
}

func allexp() ast.Node {
	ast, err := runWithCatch2(takePtrExp)
	if err == nil {
		return ast
	}
	// ast, err = runWithCatch2(takeValExp)
	// if err == nil {
	// 	return ast
	// }
	ch1 := lexer.SetCheckpoint()
	n, err := runWithCatch2(boolexp)
	if err == nil {
		ch := lexer.SetCheckpoint()
		code, _, eos := lexer.Scan()
		if eos {
			panic("unexpected eos")
		}
		lexer.GobackTo(ch)
		switch code {
		case lexer.TYPE_DIV, lexer.TYPE_MUL, lexer.TYPE_PLUS, lexer.TYPE_SUB:
		default:
			return n
		}
	}

	lexer.GobackTo(ch1)
	return exp()

}

func boolexp() (node ast.Node, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(ch)
		}
	}()
	node, err = boolean()
	if err != nil {
		return nil, err
	}
	cp := lexer.SetCheckpoint()
	co, _, eos := lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	if co == lexer.TYPE_AND || co == lexer.TYPE_OR {
		n := &ast.BoolExpNode{}
		n.Left = node
		n.Op = co
		node, err = boolexp()
		if err != nil {
			return nil, err
		}
		n.Right = node
		return n, nil
	}
	lexer.GobackTo(cp)
	return
}

func runWithCatch(f func() ast.Node) (node ast.Node, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		i := recover()
		if i != nil {
			lexer.GobackTo(ch)
			err = fmt.Errorf("%v", i)
		}
	}()
	node = f()
	return
}
func runWithCatch2(f func() (ast.Node, error)) (node ast.Node, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		i := recover()
		if i != nil {
			err = fmt.Errorf("%v", i)
		}
		if err != nil {
			lexer.GobackTo(ch)
		}
	}()
	node, err = f()
	return
}

func boolean() (node ast.Node, err error) {
	ch1 := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(ch1)
		}
	}()
	_, err = lexer.ScanType(lexer.TYPE_RES_TRUE)
	if err == nil {
		return &ast.BoolConstNode{Val: true}, nil
	}
	_, err = lexer.ScanType(lexer.TYPE_RES_FALSE)
	if err == nil {
		return &ast.BoolConstNode{Val: false}, nil
	}
	node, err = runWithCatch2(compare)
	if err == nil {
		return node, nil
	}
	n, err := runWithCatch2(takeValExp)
	if err == nil {
		return n, nil
	}

	code, _, eos := lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	switch code {
	case lexer.TYPE_NOT:
		node, err = boolean()
		if err != nil {
			return nil, err
		}
		return &ast.NotNode{Bool: node}, nil
	case lexer.TYPE_LP:
		node, err = boolexp()
		if err != nil {
			return nil, err
		}
		_, err = lexer.ScanType(lexer.TYPE_RP)
		if err != nil {
			return nil, err
		}
		return

	}

	return nil, fmt.Errorf("parse failed")
}

func compare() (node ast.Node, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(ch)
		}
	}()
	n := &ast.CompareNode{}
	n.Left = exp()
	code, _, eos := lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	switch code {
	case lexer.TYPE_EQ, lexer.TYPE_NEQ,
		lexer.TYPE_LG, lexer.TYPE_SM,
		lexer.TYPE_LEQ, lexer.TYPE_SEQ:
		n.Op = code
	default:
		return nil, fmt.Errorf("expect compare op")
	}
	n.Right = exp()
	return n, nil
}

func statementBlock() (ast.Node, error) {
	_, err := lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		return nil, err
	}
	n := statementList()
	_, err = lexer.ScanType(lexer.TYPE_RB)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func ifstatement() (n ast.Node, err error) {
	_, err = lexer.ScanType(lexer.TYPE_RES_IF)
	if err != nil {
		return nil, err
	}
	be, err := boolexp()
	if err != nil {
		return nil, err
	}
	statements, err := statementBlock()
	if err != nil {
		return nil, err
	}
	_, err = lexer.ScanType(lexer.TYPE_RES_EL)
	if err != nil {
		return &ast.IfNode{BoolExp: be, Statements: statements}, nil
	}
	elstatements, err := ifstatement()
	if err == nil {
		return &ast.IfElseNode{BoolExp: be, Statements: statements, ElSt: elstatements}, nil
	}
	elstatements, err = statementBlock()
	if err != nil {
		return nil, err
	}
	return &ast.IfElseNode{BoolExp: be, Statements: statements, ElSt: elstatements}, nil

}

func defineAndAssign() (n ast.Node, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(ch)
		}
	}()
	var id string
	_, err = lexer.ScanType(lexer.TYPE_RES_VAR)
	if err != nil {
		id, err = lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			return nil, err
		}
		_, err = lexer.ScanType(lexer.TYPE_DEAS)
		if err != nil {
			return nil, err
		}
		goto VAL
	}
	id, err = lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	_, err = lexer.ScanType(lexer.TYPE_ASSIGN)
	if err != nil {
		return nil, err
	}
VAL:
	val := allexp()
	return &ast.DefAndAssignNode{Val: val, ID: id}, nil
}

func breakST() (n ast.Node, err error) {
	_, err = lexer.ScanType(lexer.TYPE_RES_BR)
	if err != nil {
		return nil, err
	}
	empty()
	return &ast.BreakNode{}, err
}
func continueST() (n ast.Node, err error) {
	_, err = lexer.ScanType(lexer.TYPE_RES_CO)
	if err != nil {
		return nil, err
	}
	empty()
	return &ast.ContinueNode{}, nil
}

func forloop() (n ast.Node, err error) {
	_, err = lexer.ScanType(lexer.TYPE_RES_FOR)
	if err != nil {
		return nil, err
	}
	fn := &ast.ForNode{}
	def, err := defineAndAssign()
	if err == nil {
		fn.DefineAssign = def
	}
	_, err = lexer.ScanType(lexer.TYPE_SEMI)
	if err != nil {
		st, err := statementBlock()
		if err != nil {
			return nil, err
		}
		fn.Statements = st
		return fn, nil
	}
	fn.Bool, _ = boolexp()
	_, err = lexer.ScanType(lexer.TYPE_SEMI)
	if err != nil {
		return nil, err
	}
	fn.Assign, _ = assign()
	fn.Statements, err = statementBlock()
	if err != nil {
		return nil, err
	}
	return fn, nil
}

func structDef() (n ast.Node, err error) {
	_, err = lexer.ScanType(lexer.TYPE_RES_TYPE)
	if err != nil {
		return nil, err
	}
	t, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	if strings.Contains(t, ".") {
		panic("unexpected '.'")
	}
	fields := make(map[string]ast.TypeNode)
	_, err = lexer.ScanType(lexer.TYPE_RES_STRUCT)
	if err != nil {
		return nil, err
	}
	_, err = lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		return nil, err
	}
	for {
		_, err = lexer.ScanType(lexer.TYPE_RB)
		if err == nil {
			break
		}
		t, err := lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			empty()
			continue
		}
		if strings.Contains(t, ".") {
			panic("unexpected '.'")
		}
		fields[t], err = allTypes()
		if err != nil {
			panic(err)
		}
		empty()
	}
	return ast.NewStructDefNode(t, fields), nil
}

func allTypes() (n ast.TypeNode, err error) {
	ptrLevel := 0
	for {
		_, err = lexer.ScanType(lexer.TYPE_MUL)
		if err != nil {
			break
		}
		ptrLevel++
	}
	n, err = basicTypes()
	if err != nil {
		n, err = arrayTypes()
		if err != nil {
			return nil, err
		}
	}
	n.SetPtrLevel(ptrLevel)
	return
}

func arrayTypes() (n ast.TypeNode, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(ch)
		}
	}()
	var arr *ast.ArrayTypeNode = &ast.ArrayTypeNode{}
	_, err = lexer.ScanType(lexer.TYPE_LSB)
	if err != nil {
		return nil, err
	}
	t, err := lexer.ScanType(lexer.TYPE_INT)
	if err != nil {
		return nil, err
	}
	arr.Len, _ = strconv.Atoi(t)
	_, err = lexer.ScanType(lexer.TYPE_RSB)
	if err != nil {
		return nil, err
	}
	if arr == nil {
		return nil, fmt.Errorf("not array type")
	}
	tn, err := allTypes()
	if err != nil {
		return nil, err
	}
	arr.ElmType = tn
	return arr, nil

}

func basicTypes() (n ast.TypeNode, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(ch)
		}
	}()
	code, t, eos := lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	tp := []string{t}
	co, ok := lexer.IsResType(t)
	if !ok {
		if code == lexer.TYPE_VAR {
			_, err = lexer.ScanType(lexer.TYPE_DOT)
			if err == nil {
				// module
				t, err = lexer.ScanType(lexer.TYPE_VAR)
				if err != nil {
					return nil, err
				}
				tp = append(tp, t)
			}
			return &ast.BasicTypeNode{CustomTp: tp}, nil
		} else {
			return nil, fmt.Errorf("not basic type")
		}
	}
	return &ast.BasicTypeNode{ResType: co}, nil
}

func structInit() (n ast.Node, err error) {
	t, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	tp := []string{t}
	_, err = lexer.ScanType(lexer.TYPE_DOT)
	if err == nil {
		// module
		t, err = lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			return nil, err
		}
		tp = append(tp, t)
	}
	stNode := &ast.StructInitNode{
		ID:     tp,
		Fields: make(map[string]ast.Node),
	}
	_, err = lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		return nil, err
	}
	for {
		_, err = lexer.ScanType(lexer.TYPE_RB)
		if err == nil {
			break
		}
		t, err := lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			empty()
			continue
		}
		if strings.Contains(t, ".") {
			panic("unexpected '.'")
		}
		_, err = lexer.ScanType(lexer.TYPE_COLON)
		if err != nil {
			return nil, err
		}
		stNode.Fields[t] = allexp()
		_, err = lexer.ScanType(lexer.TYPE_COMMA)
		if err != nil {
			_, err = lexer.ScanType(lexer.TYPE_RB)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	return stNode, nil
}

func arrayInit() (n ast.Node, err error) {
	an := &ast.ArrayInitNode{}
	tp, err := arrayTypes()
	if err != nil {
		return nil, err
	}
	an.Type = tp
	_, err = lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		return nil, err
	}
	for {
		_, err = lexer.ScanType(lexer.TYPE_RB)
		if err == nil {
			break
		}
		_, err = lexer.ScanType(lexer.TYPE_NL)
		if err == nil {
			continue
		}
		an.Vals = append(an.Vals, allexp())
		_, err = lexer.ScanType(lexer.TYPE_COMMA)
		if err != nil {
			_, err = lexer.ScanType(lexer.TYPE_RB)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	return an, err
}

func takePtrExp() (n ast.Node, err error) {
	_, err = lexer.ScanType(lexer.TYPE_ESP)
	if err != nil {
		return nil, err
	}
	var node ast.Node
	node, err = runWithCatch2(arrayInit)
	if err == nil {
		return &ast.TakePtrNode{Node: node}, nil
	}
	node, err = runWithCatch2(structInit)
	if err == nil {
		return &ast.TakePtrNode{Node: node}, nil
	}
	node, err = runWithCatch2(varChain)
	if err != nil {
		return nil, err
	}
	return &ast.TakePtrNode{Node: node}, nil

}
func takeValExp() (n ast.Node, err error) {
	level := 0
	for {
		_, err = lexer.ScanType(lexer.TYPE_MUL)
		if err != nil {
			break
		}
		level++
	}
	var node ast.Node
	node, err = runWithCatch2(arrayInit)
	if err == nil {
		return &ast.TakeValNode{Node: node, Level: level}, nil
	}
	node, err = runWithCatch2(structInit)
	if err == nil {
		return &ast.TakeValNode{Node: node, Level: level}, nil
	}
	ch := lexer.SetCheckpoint()
	node, err = runWithCatch2(varChain)
	if err != nil {
		return nil, err
	}
	_, err = lexer.ScanType(lexer.TYPE_LP)
	if err == nil {
		lexer.GobackTo(ch)
		node = callFunc()
		return &ast.TakeValNode{Node: node, Level: level}, nil
	}
	return &ast.TakeValNode{Node: node, Level: level}, nil

}

func varChain() (n ast.Node, err error) {
	head, err := varBlock()
	if err != nil {
		return nil, err
	}
	curr := head
	for {
		_, err := lexer.ScanType(lexer.TYPE_DOT)
		if err != nil {
			break
		}
		curr.Next, err = varBlock()
		if err != nil {
			return nil, err
		}
		curr = curr.Next
	}
	return head, nil
}
func varBlock() (n *ast.VarBlockNode, err error) {
	t, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	n = &ast.VarBlockNode{
		Token: t,
	}
	for {
		_, err := lexer.ScanType(lexer.TYPE_LSB)
		if err != nil {
			break
		}
		n.Idxs = append(n.Idxs, allexp())
		_, err = lexer.ScanType(lexer.TYPE_RSB)
		if err != nil {
			return nil, err
		}
	}
	return n, nil
}

func Parse(s string) string {
	m := ir.NewModule()
	ast.AddSTDFunc(m)
	ast := ParseAST(s)
	ast.Emit(m)
	return m.String()
}
func ParseAST(s string) *ast.ProgramNode {
	lexer.SetInput(s)

	return program()
}

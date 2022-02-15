package parser

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/Chronostasys/calc/compiler/ast"
	"github.com/Chronostasys/calc/compiler/helper"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"

	"github.com/Chronostasys/calc/compiler/lexer"
)

func ParseInt(s string) (int64, *types.IntType, error) {
	base := 10
	if len(s) > 2 {
		switch s[:2] {
		case "0x":
			base = 16
			s = s[2:]
		case "0b":
			base = 2
			s = s[2:]
		case "0o":
			base = 8
			s = s[2:]

		}
	}
	bw := 8
	for {
		re, err := strconv.ParseInt(s, base, bw)
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

type Parser struct {
	imp     map[string]string
	mod     string
	scope   *ast.Scope
	lexer   *lexer.Lexer
	m       *ir.Module
	fathers map[string]bool
}

func NewParser(mod string, m *ir.Module, fathers map[string]bool) *Parser {
	p := &Parser{
		lexer:   &lexer.Lexer{},
		scope:   ast.NewGlobalScope(m),
		mod:     mod,
		m:       m,
		fathers: fathers,
	}
	p.scope.Pkgname = mod
	return p
}

func (p *Parser) assign() (n ast.Node, err error) {
	c := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(c)
		}
	}()
	level := 0
	for {
		_, err = p.lexer.ScanType(lexer.TYPE_MUL)
		if err != nil {
			break
		}
		level++
	}
	node, err := p.runWithCatch2(p.varChain)
	if err != nil {
		return nil, err
	}
	_, err = p.lexer.ScanType(lexer.TYPE_ASSIGN)
	if err != nil {
		return nil, err
	}
	r := p.allexp()
	return &ast.BinNode{
		Left:  &ast.TakeValNode{Node: node, Level: level},
		Op:    lexer.TYPE_ASSIGN,
		Right: r,
	}, nil
}

func (p *Parser) empty() ast.Node {
	_, err := p.lexer.ScanType(lexer.TYPE_NL)
	if err != nil {
		panic(err)
	}
	return &ast.EmptyNode{}
}

func (p *Parser) define() (n ast.Node, err error) {
	c := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(c)
		}
		if err == nil {
			p.empty()
		}
	}()
	_, err = p.lexer.ScanType(lexer.TYPE_RES_VAR)
	if err != nil {
		return nil, err
	}
	id, err := p.lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	tp, err := p.allTypes()
	if err != nil {
		panic(err)
	}
	return &ast.DefineNode{ID: id, TP: tp}, nil
}

func (p *Parser) statement() ast.Node {
	ast, err := p.runWithCatch2(p.continueST)
	if err == nil {
		return ast
	}
	ast, err = p.runWithCatch2(p.breakST)
	if err == nil {
		return ast
	}
	ast, err = p.runWithCatch2(p.forloop)
	if err == nil {
		return ast
	}
	ast, err = p.runWithCatch2(p.defineAndAssign)
	if err == nil {
		return ast
	}
	ast, err = p.runWithCatch2(p.ifstatement)
	if err == nil {
		return ast
	}
	ast, err = p.runWithCatch2(p.assign)
	if err == nil {
		return ast
	}
	ast, err = p.runWithCatch2(p.define)
	if err == nil {
		return ast
	}
	ast, err = p.runWithCatch2(p.returnST)
	if err == nil {
		return ast
	}
	ch := p.lexer.SetCheckpoint()
	c, t, _ := p.lexer.Scan()
	if c == lexer.TYPE_VAR {
		p.lexer.GobackTo(ch)
		cf := p.callFunc()
		p.empty()
		return cf
	} else if c == lexer.TYPE_NL {
		p.lexer.GobackTo(ch)
		return p.empty()
	}
	panic(fmt.Sprintf("parse fail %s", t))
}

func (p *Parser) statementList() ast.Node {
	n := &ast.SLNode{}
	for {
		n.Children = append(n.Children, p.statement())
		ch := p.lexer.SetCheckpoint()
		c, _, _ := p.lexer.Scan()
		p.lexer.GobackTo(ch)
		if c == lexer.TYPE_RB {
			return n
		}
	}
}

func (p *Parser) program() *ast.ProgramNode {
	n := &ast.ProgramNode{GlobalScope: p.scope}
	astnode, err := p.pkgDeclare()
	if err != nil {
		panic("missing package declareation on begining of source file")
	}
	n.PKG = astnode
	_, m := path.Split(p.mod)
	if astnode.Name != m && astnode.Name != "main" {
		panic(fmt.Errorf("bad mod %s", astnode.Name))
	}
	if astnode.Name == "main" {
		p.mod = astnode.Name
		p.scope.Pkgname = astnode.Name
	}
	for {
		_, err := p.lexer.ScanType(lexer.TYPE_NL)
		if err != nil {
			break
		}
	}
	imp, _ := p.importStatement()
	n.Imports = imp
	p.imp = map[string]string{}
	if imp != nil {
		p.imp = imp.Imports
		for _, v := range p.imp {
			if strings.Index(v, calcmod) == 0 {
				// sub module of mod
				pa := path.Join(maindir, v[len(calcmod):])
				if p.fathers[v] {
					panic(fmt.Sprintf("found loop referencing in %s. refmap: %v", v, p.fathers))
				}
				ParseModule(pa, v, p.m, p.fathers)
			} else {
				// TODO external module
				panic("not impl")
			}
		}
	}
	for {
		ch := p.lexer.SetCheckpoint()
		c, _, eos := p.lexer.Scan()
		if c == lexer.TYPE_NL {
			continue
		}
		p.lexer.GobackTo(ch)
		if eos {
			break
		}
		ast, err := p.runWithCatch2(p.typeDef)
		if err == nil {
			n.Children = append(n.Children, ast)
			continue
		}
		ast, err = p.runWithCatch2(p.define)
		if err == nil {
			n.Children = append(n.Children, ast)
			continue
		}
		ast, err = p.runWithCatch2(p.defineAndAssign)
		if err == nil {
			n.Children = append(n.Children, ast)
			continue
		}

		n.Children = append(n.Children, p.function())
	}
	return n
}

func (p *Parser) allexp() ast.Node {
	ast, err := p.runWithCatch2(p.takePtrExp)
	if err == nil {
		return ast
	}
	n, err := p.boolexp()
	if err != nil {
		panic(err)
	}
	return n

}

func (p *Parser) runWithCatch(f func() ast.Node) (node ast.Node, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		i := recover()
		if i != nil {
			p.lexer.GobackTo(ch)
			err = fmt.Errorf("%v", i)
		}
	}()
	node = f()
	return
}
func (p *Parser) runWithCatch2(f func() (ast.Node, error)) (node ast.Node, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		i := recover()
		if i != nil {
			err = fmt.Errorf("%v", i)
		}
		if err != nil {
			p.lexer.GobackTo(ch)
		}
	}()
	node, err = f()
	return
}
func (p *Parser) statementBlock() (ast.Node, error) {
	_, err := p.lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		return nil, err
	}
	n := p.statementList()
	_, err = p.lexer.ScanType(lexer.TYPE_RB)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (p *Parser) ifstatement() (n ast.Node, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_IF)
	if err != nil {
		return nil, err
	}
	be, err := p.boolexp()
	if err != nil {
		return nil, err
	}
	statements, err := p.statementBlock()
	if err != nil {
		return nil, err
	}
	_, err = p.lexer.ScanType(lexer.TYPE_RES_EL)
	if err != nil {
		return &ast.IfNode{BoolExp: be, Statements: statements}, nil
	}
	elstatements, err := p.ifstatement()
	if err == nil {
		return &ast.IfElseNode{BoolExp: be, Statements: statements, ElSt: elstatements}, nil
	}
	elstatements, err = p.statementBlock()
	if err != nil {
		return nil, err
	}
	return &ast.IfElseNode{BoolExp: be, Statements: statements, ElSt: elstatements}, nil

}

func (p *Parser) defineAndAssign() (n ast.Node, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(ch)
		}
	}()
	var id string
	_, err = p.lexer.ScanType(lexer.TYPE_RES_VAR)
	if err != nil {
		id, err = p.lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			return nil, err
		}
		_, err = p.lexer.ScanType(lexer.TYPE_DEAS)
		if err != nil {
			return nil, err
		}
		goto VAL
	}
	id, err = p.lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	_, err = p.lexer.ScanType(lexer.TYPE_ASSIGN)
	if err != nil {
		return nil, err
	}
VAL:
	val := p.allexp()
	return &ast.DefAndAssignNode{Val: val, ID: id}, nil
}

func (p *Parser) breakST() (n ast.Node, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_BR)
	if err != nil {
		return nil, err
	}
	p.empty()
	return &ast.BreakNode{}, err
}
func (p *Parser) continueST() (n ast.Node, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_CO)
	if err != nil {
		return nil, err
	}
	p.empty()
	return &ast.ContinueNode{}, nil
}

func (p *Parser) forloop() (n ast.Node, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_FOR)
	if err != nil {
		return nil, err
	}
	fn := &ast.ForNode{}
	def, err := p.defineAndAssign()
	if err == nil {
		fn.DefineAssign = def
	}
	_, err = p.lexer.ScanType(lexer.TYPE_SEMI)
	if err != nil {
		st, err := p.statementBlock()
		if err != nil {
			return nil, err
		}
		fn.Statements = st
		return fn, nil
	}
	fn.Bool, _ = p.boolexp()
	_, err = p.lexer.ScanType(lexer.TYPE_SEMI)
	if err != nil {
		return nil, err
	}
	fn.Assign, _ = p.assign()
	fn.Statements, err = p.statementBlock()
	if err != nil {
		return nil, err
	}
	return fn, nil
}

func (p *Parser) allTypes() (n ast.TypeNode, err error) {
	ptrLevel := 0
	for {
		_, err = p.lexer.ScanType(lexer.TYPE_MUL)
		if err != nil {
			break
		}
		ptrLevel++
	}
	n, err = p.structType()
	if err == nil {
		goto END
	}
	n, err = p.interfaceType()
	if err == nil {
		goto END
	}
	n, err = p.basicTypes()
	if err != nil {
		n, err = p.arrayTypes()
		if err != nil {
			return nil, err
		}
	}
END:
	n.SetPtrLevel(ptrLevel)
	return
}

func (p *Parser) arrayTypes() (n ast.TypeNode, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(ch)
		}
	}()
	var arr *ast.ArrayTypeNode = &ast.ArrayTypeNode{}
	_, err = p.lexer.ScanType(lexer.TYPE_LSB)
	if err != nil {
		return nil, err
	}
	t, err := p.lexer.ScanType(lexer.TYPE_INT)
	if err != nil {
		return nil, err
	}
	arr.Len, _ = strconv.Atoi(t)
	_, err = p.lexer.ScanType(lexer.TYPE_RSB)
	if err != nil {
		return nil, err
	}
	if arr == nil {
		return nil, fmt.Errorf("not array type")
	}
	tn, err := p.allTypes()
	if err != nil {
		return nil, err
	}
	arr.ElmType = tn
	return arr, nil

}

func (p *Parser) basicTypes() (n ast.TypeNode, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(ch)
		}
	}()
	code, t, eos := p.lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	tp := []string{t}
	co, ok := lexer.IsResType(t)
	if !ok {
		if code == lexer.TYPE_VAR {
			_, err = p.lexer.ScanType(lexer.TYPE_DOT)
			if err == nil {
				// module
				t, err = p.lexer.ScanType(lexer.TYPE_VAR)
				if err != nil {
					return nil, err
				}
				tp = append(tp, t)
				tp[0] = p.imp[tp[0]]
			}
			generic, _ := p.genericCallParams()
			return &ast.BasicTypeNode{CustomTp: tp, Generics: generic, Pkg: p.mod}, nil
		} else {
			return nil, fmt.Errorf("not basic type")
		}
	}
	return &ast.BasicTypeNode{ResType: co}, nil
}

func (p *Parser) structInit() (n ast.Node, err error) {
	tp, err := p.allTypes()
	if err != nil {
		return nil, err
	}
	stNode := &ast.StructInitNode{
		TP:     tp,
		Fields: make(map[string]ast.Node),
	}
	_, err = p.lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		return nil, err
	}
	for {
		_, err = p.lexer.ScanType(lexer.TYPE_RB)
		if err == nil {
			break
		}
		t, err := p.lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			p.empty()
			continue
		}
		if strings.Contains(t, ".") {
			panic("unexpected '.'")
		}
		_, err = p.lexer.ScanType(lexer.TYPE_COLON)
		if err != nil {
			return nil, err
		}
		stNode.Fields[t] = p.allexp()
		_, err = p.lexer.ScanType(lexer.TYPE_COMMA)
		if err != nil {
			_, err = p.lexer.ScanType(lexer.TYPE_RB)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	return stNode, nil
}

func (p *Parser) arrayInit() (n ast.Node, err error) {
	an := &ast.ArrayInitNode{}
	tp, err := p.arrayTypes()
	if err != nil {
		return nil, err
	}
	an.Type = tp
	_, err = p.lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		return nil, err
	}
	for {
		_, err = p.lexer.ScanType(lexer.TYPE_RB)
		if err == nil {
			break
		}
		_, err = p.lexer.ScanType(lexer.TYPE_NL)
		if err == nil {
			continue
		}
		an.Vals = append(an.Vals, p.allexp())
		_, err = p.lexer.ScanType(lexer.TYPE_COMMA)
		if err != nil {
			_, err = p.lexer.ScanType(lexer.TYPE_RB)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	return an, err
}

func (p *Parser) takePtrExp() (n ast.Node, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_ESP)
	if err != nil {
		return nil, err
	}
	var node ast.Node
	node, err = p.runWithCatch2(p.arrayInit)
	if err == nil {
		return &ast.TakePtrNode{Node: node}, nil
	}
	node, err = p.runWithCatch2(p.structInit)
	if err == nil {
		return &ast.TakePtrNode{Node: node}, nil
	}
	node, err = p.runWithCatch2(p.varChain)
	if err != nil {
		return nil, err
	}
	return &ast.TakePtrNode{Node: node}, nil

}
func (p *Parser) takeValExp() (n ast.Node, err error) {
	level := 0
	for {
		_, err = p.lexer.ScanType(lexer.TYPE_MUL)
		if err != nil {
			break
		}
		level++
	}
	var node ast.Node
	defer func() {
		if err != nil {
			return
		}
		if level == 0 {
			n = node
		}
	}()
	node, err = p.runWithCatch2(p.arrayInit)
	if err == nil {
		return &ast.TakeValNode{Node: node, Level: level}, nil
	}
	node, err = p.runWithCatch2(p.structInit)
	if err == nil {
		return &ast.TakeValNode{Node: node, Level: level}, nil
	}

	node, err = p.runWithCatch(p.callFunc)
	if err == nil {
		return &ast.TakeValNode{Node: node, Level: level}, nil
	}
	node, err = p.runWithCatch2(p.varChain)
	if err != nil {
		return nil, err
	}
	return &ast.TakeValNode{Node: node, Level: level}, nil

}

func (p *Parser) varChain() (n ast.Node, err error) {
	head, err := p.varBlock()
	if err != nil {
		return nil, err
	}
	pkg, ok := p.imp[head.Token]
	if ok {
		head.Token = pkg
	}
	curr := head
	for {
		_, err := p.lexer.ScanType(lexer.TYPE_DOT)
		if err != nil {
			break
		}
		curr.Next, err = p.varBlock()
		if err != nil {
			return nil, err
		}
		curr = curr.Next
	}
	return head, nil
}
func (p *Parser) varBlock() (n *ast.VarBlockNode, err error) {
	t, err := p.lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	n = &ast.VarBlockNode{
		Token: t,
	}
	for {
		_, err := p.lexer.ScanType(lexer.TYPE_LSB)
		if err != nil {
			break
		}
		n.Idxs = append(n.Idxs, p.allexp())
		_, err = p.lexer.ScanType(lexer.TYPE_RSB)
		if err != nil {
			return nil, err
		}
	}
	return n, nil
}

func (p *Parser) nilExp() (n ast.Node, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_NIL)
	if err != nil {
		return nil, err
	}
	return &ast.NilNode{}, nil
}

func (p *Parser) Parse(s string) string {
	m := ir.NewModule()
	ast.AddSTDFunc(m, p.scope)
	ast := p.ParseAST(s)
	ast.Emit(m)
	return m.String()
}
func (p *Parser) ParseAST(s string) *ast.ProgramNode {
	defer func() {
		err := recover()
		if err != nil {
			p.lexer.PrintCurrent()
			panic(err)
		}
	}()
	p.lexer.SetInput(s)

	return p.program()
}

func getModule(dir string) string {
	for i := 0; i < 20; i++ {
		_, err := os.Stat(path.Join(dir, "calc.mod"))
		if err == nil {
			// path/to/whatever does not exist
			bs, err := ioutil.ReadFile(path.Join(dir, "calc.mod"))
			if err != nil {
				panic(err)
			}
			str := string(bs)
			mod := ""
			fmt.Sscanf(str, "module %s", &mod)
			maindir = dir
			return mod
		}
		if os.IsNotExist(err) {
			dir = path.Join(dir, "..")
			continue
		}
		panic(err)
	}
	panic("cannot find mod file")
}

var calcmod, maindir string
var startMap = map[string]bool{}
var mu = &sync.Mutex{}

func ParseDir(dir string) *ir.Module {
	calcmod = getModule(dir)
	m := ir.NewModule()
	mod := "github.com/Chronostasys/calc/runtime"
	pa := path.Join(maindir, mod[len(calcmod):])
	ParseModule(pa, mod, m, map[string]bool{})
	p1 := ParseModule(dir, "main", m, map[string]bool{})
	ast.AddSTDFunc(m, p1.GlobalScope)
	return m
}
func ParseModule(dir, mod string, m *ir.Module, fathers map[string]bool) *ast.ProgramNode {
	mu.Lock()
	if startMap[mod] {
		mu.Unlock()
		return nil
	}
	startMap[mod] = true
	mu.Unlock()
	tmpm := ir.NewModule()
	c, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	nodes := []*ast.ProgramNode{}
	fileNum := 0
	nodeCh := make(chan *ast.ProgramNode)
	errch := make(chan error)
	for _, v := range c {
		newF := map[string]bool{}
		for k, v := range fathers {
			newF[k] = v
		}
		newF[mod] = true
		if !v.IsDir() {
			name := v.Name()
			sp := helper.SplitLast(name, ".")
			if !(len(sp) == 2 && sp[1] == "calc") {
				continue
			}
			fileNum++
			go func() {
				bs, err := ioutil.ReadFile(path.Join(dir, name))
				if err != nil {
					errch <- err
				}
				str := string(bs)
				p := NewParser(mod, m, newF)
				nodeCh <- p.ParseAST(str)
			}()
		}
	}
	if fileNum == 0 {
		log.Fatalln("cannot find source file at", dir)
	}
	for i := 0; i < fileNum; i++ {
		select {
		case err := <-errch:
			panic(err)
		case node := <-nodeCh:
			nodes = append(nodes, node)
		}
	}

	p := ast.Merge(nodes...)
	ast.AddSTDFunc(tmpm, p.GlobalScope)
	emitMu.Lock()
	defer emitMu.Unlock()
	ast.ScopeMap[mod] = p.GlobalScope
	p.Emit(m)
	return p

}

var emitMu = &sync.Mutex{}

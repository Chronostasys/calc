package parser

// func (p *Parser) idxReload() (n ast.Node, err error) {
// 	_, err = p.lexer.ScanType(lexer.TYPE_RES_OP)
// 	if err != nil {
// 		return nil, err
// 	}
// 	_, err = p.lexer.ScanType(lexer.TYPE_LSB)
// 	if err != nil {
// 		return nil, err
// 	}
// 	_, err = p.lexer.ScanType(lexer.TYPE_RSB)
// 	if err != nil {
// 		return nil, err
// 	}
// 	_, err = p.lexer.ScanType(lexer.TYPE_LP)
// 	if err != nil {
// 		return nil, err
// 	}
// 	src, err := p.extFuncParam()
// 	if err != nil {
// 		return nil, err
// 	}
// 	idx := p.funcParam()
// 	_, err = p.lexer.ScanType(lexer.TYPE_RP)
// 	if err != nil {
// 		return nil, err
// 	}
// 	tp, err := p.allTypes()
// 	if err != nil {
// 		return nil, err
// 	}
// 	sts, err := p.statementBlock()
// 	if err != nil {
// 		return nil, err
// 	}

// }

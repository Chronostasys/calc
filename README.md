# Calculator
Try to build a simple calculator, with lexer and recursive descent parser.


# Rules
```
call_func: var LP (RP|(var(COMMA var)* RP))
function: FN->FUNC var FPS TYPE LB SL RB
func_params: FPS->LP (RP|(FP(COMMA FP)* RP))
func_param: FP->var TYPE
statemnt_list: SL->S|S NL SL
statement: S->EM|D|A
empty: EM->
define: D->VAR var TYPE
asssign: A->var ASSIGN E
exp: E->F|F((ADD|MIN)F)*
factor: F->S|S((MUL|DIV)S)*
symbol: S->N|ADD N|MIN N
number: N->n|LP S RP|var
```

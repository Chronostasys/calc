# Calculator
Try to build a simple calculator, with lexer and recursive descent parser.


# Rules
```
statemnt_list: SL->S|S NL SL
statement: S->EM|D|A
empty: EM->
define: D->VAR var TYPE
asssign: A->var ASSIGN E
exp: E->F|F((ADD|MIN)F)*
factor: F->S|S((MUL|DIV)S)*
symbol: S->I|ADD I|MIN I
interger: I->i|LP S RP|var
```

# Calculator
Try to build a simple calculator, with lexer and recursive descent parser.


# Rules
```
exp: E->F|F((ADD|MIN)F)*
factor: F->S|S((MUL|DIV)S)*
symbol: S->I|ADD I|MIN I
interger: I->i|LP S RP
```

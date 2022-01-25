# Calculator
Try to build a simple calculator, with lexer and recursive descent parser.


# Rules
```
exp: S->F|F((ADD|MIN)F)*
factor: F->I|I((MUL|DIV)I)*
interger: I->i|LP S RP
```

# Calculator
Try to build a simple calculator, with lexer and parser.


# Lexer-Rules
```
S -> N
N -> N+N|i
```

regex: $S=i(\\'+\\'i)^*$

DFA:
![DFA](2022-01-18-23-38-26.png)

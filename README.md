# Calculator
~~Try to build a simple calculator, with lexer and recursive descent parser~~.

现在它是一个小编译器了  
在项目根目录`make`可以编译[test.calc](cmd/test.calc)的ir文件、可执行文件和汇编  
`make compiler`可以编译出编译器  
因为是计算器基础上改的，所以暂时的文件后缀是`.calc`  

## 工具链
- clang (llvm 12.0)
- golang 1.17.1
## Rules
```
program: P->(FN|NL)+
call_func: CF->var LP (RP|(E(COMMA AE)* RP))
function: FN->FUNC var FPS TYPE LB SL RB
func_params: FPS->LP (RP|(FP(COMMA FP)* RP))
func_param: FP->var TYPE
statemnt_list: SL->S|S NL SL
statement: S->EM|D|A|R|CF
return: R->RET|RET AE
empty: EM->
define: D->VAR var TYPE
asssign: A->var ASSIGN AE
all_exp: AE->E|BE
exp: E->F|F((ADD|MIN)F)*
factor: F->S|S((MUL|DIV)S)*
symbol: S->N|ADD N|MIN N
number: N->n|LP E RP|var|CF


bool_exp: BE->B|B AND B|B OR B
boolean: B->TRUE|FALSE|exp EQ exp|exp NOT EQ exp|NOT B|LP BE RP
```

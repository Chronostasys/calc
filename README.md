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
program: P->(FN|NL|T|INTER|D)+
call_func: CF->VC GPC? LP (RP|(E(COMMA AE)* RP))
generic_params: GP->SM TYPE (COMMA var)* LG
generic_call_params: GPC->SM TYPE (COMMA TYPE)* LG
function: FN->FUNC var GP? FPS TYPE SB
func_params: FPS->LP (RP|(EFP? FP(COMMA FP)* RP))
ext_func_param: EFP->THIS FP
func_param: FP->var TYPE
statemnt_list: SL->S+
statement: S->CS|BS|EM|D|A|R|(CF NL)|I|(DA NL)
return: R->RET|(RET AE)
empty: EM->NL
define: D->VAR var TYPE NL
all_types: TYPE->MUL*  BTYPE|AT
basic_types: BTYPE->tp
array_types: AT->LSB n RSB TYPE
asssign: A->MUL* VC ASSIGN AE
all_exp: AE->E|BE|TPE|TVE
exp: E->F|F((ADD|MIN)F)*
factor: F->S|S((MUL|DIV)S)*
symbol: S->N|((ADD|MIN) N)
number: N->n|(LP E RP)|TVE
bool_exp: BE->B|(B (AND|OR) BE)
boolean: B->TRUE|FALSE|C|(NOT B)|(LP BE RP)|TVE
compare_exp: C->(E|NE) (EQ|NEQ|LG|SM|LEQ|SEQ) (E|NE)
statement_block:SB->LB SL RB NL
def_ass: DA->var DEFA E|VAR var ASSIGN E
if_st: I->IF BE SB((EL SB|I)?)
for_st: F->FOR (DA? SEMI BE SEMI A?)? SB
break_statement: BS->BR NL
continue_statement: CS->CT NL
struct_def: T->TP var STRUCT LB ((var TYPE NL)|NL)* RB
struct_init_exp: SI->(var LB ((var COLON AE COMMA)|NL)* RB)
array_init_exp: AI->AT LB ((AE COMMA)|NL)* RB
take_ptr_exp: TPE->ESP AI|SI|VC
take_val_exp: TVE->MUL* AI|SI|VC|CF
var_chain: VC->VB (DOT VB)*
var_block: VB->var (LSB AE RSB)*
interface_def: INTER->TP var INTERFACE LB ((var FPS TYPE NL)|NL)* RB
null_exp: NE->NIL
```

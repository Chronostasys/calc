# Calc
`calc`是一个类似`golang`的编程语言  
在项目根目录`make`可以编译[test](test)目录模块的ir文件、可执行文件和汇编  
`make compiler`可以编译出编译器  
因为是计算器基础上改的，所以暂时的文件后缀是`.calc`  

## 早期开发
本项目正处于早期开发阶段，很多功能尚不完善。  
进度和计划可以查看[Roadmap](docs/roadmap.md)

## 和golang主要区别
- 支持泛型，是尖括号
- 不能多返回值

## 工具链
- clang (llvm 12.0+) windows要使用[llvm mingw](https://github.com/mstorsjo/llvm-mingw/releases)
- golang 1.17.1
## 语法规则
```
program: P->PD NL* IS? (FN|NL|T|D|DA)+
call_func: CF->VC GPC? LP (RP|(E(COMMA AE)* RP)) (DOT CF|VC)*
generic_params: GP->SM var (COMMA var)* LG
generic_call_params: GPC->SM TYPE (COMMA TYPE)* LG
function: FN->FUNC var GP? FPS TYPE ASYNC? SB
func_params: FPS->LP (RP|(EFP? FP(COMMA FP)* RP))
ext_func_param: EFP->THIS FP
func_param: FP->var TYPE
statemnt_list: SL->S+
statement: S->CS|BS|EM|D|A|R|(CF NL)|I|(DA NL)|YI
return: R->RET|(RET AE)
empty: EM->NL
yield: YI->YIELD AE? NL
define: D->VAR var TYPE NL

inline_func: IFUN->FT ASYNC?  SB


all_types: TYPE->MUL*  BTYPE|AT|ST|IT
basic_types: BTYPE->tp GPC?
array_types: AT->LSB n RSB TYPE
func_types: FT->FUNC FPS TYPE
type_def: T->TP var GP TYPE
struct_type: ST->STRUCT LB ((var TYPE NL)|NL)* RB
interface_type: IT->INTERFACE LB ((var FPS TYPE NL)|NL)* RB
asssign: A->MUL* VC ASSIGN AE

all_exp: AE->BE|TPE|IFUNC|(AWAIT AE) 
exp: E->AF ((SHL|SHR) AF)*
bool_exp: BE->BO ((AND|OR) BE)?
bit_op: BO->C ((BO|ESP|XOR) C)*
compare_exp: C->(B) ((EQ|NEQ|LG|SM|LEQ|SEQ) (B))*
boolean: B->TRUE|FALSE|E|NE|(LP BE RP)|NOT B
added_factor: AF->F((ADD|MIN)F)*
factor: F->S|S((MUL|DIV|PS)S)*
symbol: S->N|((ADD|MIN) N)
number: N->n|(LP E RP)|TVE|SE


statement_block:SB->LB SL RB NL
def_ass: DA->var DEFA E|VAR var ASSIGN E
if_st: I->IF BE SB((EL SB|I)?)
for_st: F->FOR (DA? SEMI BE SEMI A?)? SB
break_statement: BS->BR NL
continue_statement: CS->CT NL
struct_init_exp: SI->(var LB ((var COLON AE COMMA)|NL)* RB)
array_init_exp: AI->AT LB ((AE COMMA)|NL)* RB
take_ptr_exp: TPE->ESP AI|SI|VC
take_val_exp: TVE->MUL* AI|SI|VC|CF
var_chain: VC->VB (DOT VB)*
var_block: VB->var (LSB AE RSB)*
null_exp: NE->NIL
pkg_declare: PD->PKG var
string_exp: SE->str
import_statement: IS->imp str|imp LP ((SE var?)|NL)* RP

idx_op_reload: IOR->OP LSB RSB LP EFP COMMA FP RP TYPE SB
```

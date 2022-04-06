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
- 采用无栈协程

## 工具链
- clang (llvm 12.0+) windows要使用[llvm mingw](https://github.com/mstorsjo/llvm-mingw/releases)
- golang 1.17.1
- libuv
- bdwgc

## Installation
clone本项目，在项目目录执行./install脚本能在大部分Debian Linux系统上安装calc编译器
```sh
git clone https://github.com/Chronostasys/calc.git
cd calc
chmod +x ./install.sh
./install.sh
calcc -?
```


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
statement: S->CS|BS|EM|D|A|R|(CF NL)|I|(DA NL)|YI|(AWAIT AE)
return: R->RET|(RET AE)
empty: EM->NL
yield: YI->YIELD AE? NL
define: D->VAR var TYPE NL

inline_func: IFUN->FT ASYNC?  SB


all_types: TYPE->MUL*  BTYPE|AT|ST|IT
basic_types: BTYPE->tp GPC?
array_types: AT->LSB n? RSB TYPE
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

## Examples

[test](test/)文件夹中有很多例子  

### Hello World

```
package main

func main() void {
    a := "hello world"
    a.PrintLn()
    return
}

```

### Functions
```
package xxx

func Foo() void {
    b := &Boo{}
    b.FooOfBoo()
    return
}

type Boo struct {
}

func FooOfBoo(this boo *Boo) void {
    return
}

```
### Generics(泛型)
[泛型链表](runtime/linkedlist/linkedlist.calc)
```
package linkedlist

type Node<T> struct {
    val T
    next *Node<T>
    prev *Node<T>
}

type List<T> struct {
    first *Node<T>
    tail *Node<T>
    len int
}

func UnShift<T>(this li *List<T>, t T) void {
    li.len = li.len + 1
    n := &Node<T>{val:t}
    if li.first == nil {
        li.first = n
        li.tail = n
        return
    }
    n.next = li.first
    li.first.prev = n
    li.first = n
    return
}


func Push<T>(this li *List<T>, t T) void {
    li.len = li.len + 1
    n := &Node<T>{val:t}
    if li.first==nil {
        li.first = n
        li.tail = n
        return
    }

    li.tail.next = n
    n.prev = li.tail
    li.tail = n
    return
}
func Len<T>(this li *List<T>) int {
    return li.len
}

func IndexOp<T>(this li *List<T>, index int) T {
    i := 0
    var n *Node<T>
    for n = li.first;i!=index;n=n.next {
        i=i+1
    }
    return n.val
}

func Shift<T>(this li *List<T>) T {
    n := li.first
    val := n.val
    li.remove(n)
    return val
}
func Pop<T>(this li *List<T>) T {
    n := li.tail
    li.remove(n)
    return n.val
}



func remove<T>(this li *List<T>, n *Node<T>) void {
    li.len = li.len - 1
    if n.prev==nil&&n.next==nil {// 唯一一个元素被删除
        li.first = nil
        li.tail = nil
    } else if n.prev==nil {// head被删除
        li.first = n.next
        n.next.prev = nil
    } else if n.next==nil {// tail被删除
        li.tail = n.prev
        li.tail.next = nil
    } else {
        n.prev.next = n.next
        n.next.prev = n.prev
    }
    return
}



func New<T>() *List<T> {
    return &List<T>{}
}

```

### 异步操作和协程
协程包是`"github.com/Chronostasys/calc/runtime/coro"`

[tcp echo server](test/tcpserver/server.calc)

```
package main

import (
    "github.com/Chronostasys/calc/runtime/coro"
    "github.com/Chronostasys/calc/runtime/libuv"
    "github.com/Chronostasys/calc/runtime/strings"
)

func getchar() byte

func main() void {
    libuv.TCPListen("0.0.0.0",8888,func (server libuv.UVTcp, status int32) void {
        s := "new tcp conn"
        s.PrintLn()
        jobf := func () coro.Task<int> async {
            for {
                buf := await server.ReadBufAsync(1)
                ss := strings.NewStr(buf.Data,buf.Len)
                ss.Print()
                re := await server.WriteBufAsync(ss)
                if re !=0 {
                    sss := "write failed"
                    sss.PrintLn()
                    printIntln(re)
                }
            }
            return 0
        }
        jobf()
        return
    })
    s1 := "tcp echo server started at 0.0.0.0:8888"
    s1.PrintLn()
    getchar()
    return
}
```
可以使用netcat进行连接测试
```
nc 127.0.0.1 8888
```

TODO

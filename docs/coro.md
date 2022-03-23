# 协程
calc的协程基于generator  

## generator
自动生成的generator结构：  
```
|===============
|接口`StateMachine`
|各种内部变量      
|下一个block地址    
|返回值             
|===============

```

## api（草稿）
用线程池的方法是`Queue`函数，该函数接受一个generator。  
每个异步方法都可以被编译成generator  
在一个await 别的异步操作的方法中，await那一步就是一个suspend，并且返回false
（也就是await之后就移出执行队列了），在await的异步操作结束之后再重新入队列  
如果异步函数返回的类型是`T`，则实际返回类型签名为`Generator<T>`  
重新入队：需要某种链表一样的结构把它们串起来  

## 工作流程举例
假设我们有以下的异步方法：  
```calc
func DojobAsync() Task<void> async {
    i := await RunTask<int>(func () int {
        Sleep(1)
        return 100
    })
    printIntln(i)
    return
}

func main() void {
    DojobAsync()
    Sleep(3)
    return
}

```
首先，main方法里的对异步方法的调用会被变成QueueTask(DoJobAsync())  
对于异步方法，和generator一样会被编译成状态机。任何的await就是suspend点  
在await点：
- 首先运行生成状态机的函数获取状态机
- 将状态机头部的StateMachine值设为本方法的状态机
- 用QueTask将状态机加入队列
- 这次stepnext返回false（这样这个状态机会出队列）
- 生成下一个block
- 在下一个block中将await的结果赋值（通过generator的GetCurrent获取）

QueueTask的每个状态机运行完之后，检查该状态机的头部StateMachine是否是nil
若不是则将该值用QueueTask入队列  


## 异步io
预计使用https://think-async.com/Asio/  
异步io生成的异步状态机会构造将老函数状态机重新入队列的闭包函数作为异步操作的回调函数


## memory leak

之前async func出现过一次严重的memory leak，详情见[leak.md](leak.md)
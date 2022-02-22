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

```calc
func QueueTask(s StateMachine) void {

}

```

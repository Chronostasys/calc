# 闭包

闭包使用`llvm.init.trampoline`和`llvm.adjust.trampoline`实现，见https://llvm.org/docs/LangRef.html#llvm-init-trampoline-intrinsic  
思路见https://stackoverflow.com/questions/8706998/how-to-efficiently-implement-closures-in-llvm-ir  

## 注意事项

如果闭包里有闭包，那么外部闭包要引用内部闭包。否则内部闭包会过早被回收？


# Garbadge collector
gc设计
## 内存布局
所有的struct和array多加6个隐藏的field，放在开头
```
|=============
|pointer    i1      // 是否是指针
|refs       [x]i8   // 引用类型字段的index，数组类型这里为元素的refs的值
|reflen     i8      // refs的长度
|screfs     [x]i8   // 字段为包含引用类型字段的结构体的index，数组类型这里为空
|screflen   i8      // screfs的长度，如果数组元素是值类型的struct，这里填-1
|color      i8      // 0: white 1: grey 2: black
|len        int     // 结构体类型这里是-1，数组类型这里是长度
|felds/elements...  // 用户定义的字段或者数组元素
```



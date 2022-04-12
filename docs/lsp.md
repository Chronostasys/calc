# Language server

首先编译器要能容忍错误  
那么要改现有代码：  
- parser永不panic，无法parse的情况生成error node
- 由于现有的编译器编译过程中会大量试错，所以很多地方panic或者返回err代码不能变，error node只在statement list尝试完全部失败的时候产生


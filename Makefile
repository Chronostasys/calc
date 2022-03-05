.PHONY: compiler

all: compiler test-ir test-asm test-exe

test-ir:
	cd compiler && compiler.exe -d ../test -o ../test/out.ll

test-asm:
	cd test && clang -S out.ll -o test.asm

test-exe:
	cd test && copy ..\bin\win\bdwgc\*.* *.* && copy ..\bin\win\libuv\*.* *.* && clang out.ll libgc.dll.a uv.lib uvutil.a -static-libgcc -static-libstdc++ -lpthread  -o test.exe
compiler:
	cd compiler && go build -o compiler.exe main.go

gc-dependency:
	cd bdwgc && git clone git://github.com/ivmai/libatomic_ops.git

gc-windows:
	cd bdwgc && nmake -f NT_MAKEFILE cpu=AMD64 nodebug=1 && copy gc.lib  ..\compiler\gc.lib && copy gc.dll ..\compiler\gc.dll
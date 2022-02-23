.PHONY: compiler

all: compiler test-ir test-asm test-exe

test-ir:
	cd compiler && compiler.exe -d ../test -o ../test/out.ll

test-asm:
	cd test && clang -S out.ll -o test.asm

test-exe:
	cd test && copy ..\compiler\gc.lib  gc.lib && copy ..\compiler\gc.dll gc.dll  && clang out.ll gc.lib -static-libgcc -static-libstdc++ -o test.exe
compiler:
	cd compiler && go build -o compiler.exe main.go

gc-dependency:
	cd bdwgc && git clone git://github.com/ivmai/libatomic_ops.git

gc-windows:
	cd bdwgc && nmake -f NT_MAKEFILE cpu=AMD64 nodebug=1 && copy gc.lib  ..\compiler\gc.lib && copy gc.dll ..\compiler\gc.dll
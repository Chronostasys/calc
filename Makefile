all: test-ir test-asm test-exe compiler

test-ir:
	cd cmd && go run main.go

test-asm:
	cd cmd && clang -S test.ll -o test.asm

test-exe:
	cd cmd && clang test.ll gc.lib -o test.exe
compiler:
	cd cmd && go build -o compiler.exe main.go

gc-dependency:
	cd bdwgc && git clone git://github.com/ivmai/libatomic_ops.git

gc-windows:
	cd bdwgc && nmake -f NT_MAKEFILE cpu=AMD64 nodebug=1 && copy gc.lib  ..\cmd\gc.lib && copy gc.dll ..\cmd\gc.dll
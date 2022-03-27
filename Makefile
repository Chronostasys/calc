.PHONY: compiler

all: uv-helper compiler test-ir test-asm test-exe

uv-helper:
	cd libuv_helper && make

test-ir:
	cd compiler && calccf.exe -d ../test -o ../test/out.ll

test-tcp:
	cd compiler && calccf.exe -d ../test/tcpserver -o ../test/out.ll

test-asm:
	cd test && clang -S out.ll -o test.asm

test-exe:
	cd test && copy ..\bin\win\bdwgc\*.* *.* && copy ..\bin\win\libuv\*.* *.* && clang out.ll libgc.dll.a uv.lib uvutil.a -static-libgcc -static-libstdc++ -lpthread  -o test.exe
compiler:
	cd compiler && go build -o calccf.exe main.go && copy calccf.exe ..\bin\win\calccf.exe

compiler-linux:
	cd compiler && go build -o calccf main.go && sudo cp calccf /usr/local/bin/calccf

install: compiler-linux
	sudo cp bin/linux/calcc.sh /usr/local/bin/calcc

gc-dependency:
	cd bdwgc && git clone git://github.com/ivmai/libatomic_ops.git

gc-windows:
	cd bdwgc && nmake -f NT_MAKEFILE cpu=AMD64 nodebug=1
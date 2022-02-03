all: test-ir test-asm test-exe compiler

test-ir:
	cd cmd && go run main.go

test-asm:
	cd cmd && clang -S test.ll -o test.asm

test-exe:
	cd cmd && clang test.ll -o test.exe
compiler:
	cd cmd && go build -o compiler.exe main.go
test-cmd:
	cd cmd && go run main.go && clang -S test.ll -o test.asm && clang test.ll -o test.exe
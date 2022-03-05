.PHONY: all
all:
	clang -I . -c -Wall .\uvutil.c ..\bin\win\bdwgc\libgc.dll.a && llvm-ar rc uvutil.a uvutil.o && copy uvutil.a ..\bin\win\libuv\uvutil.a
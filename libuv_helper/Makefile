.PHONY: all
all:
	clang -I . -c -Wall .\uvutil.c ..\bin\win\bdwgc\libgc.dll.a -pthread && llvm-ar rc uvutil.a uvutil.o && copy uvutil.a ..\bin\win\libuv\uvutil.a

linux:
	clang -I . -c  -Wall ./uvutil.c && ar r uvutil.a uvutil.o && cp uvutil.a /usr/local/lib && cp uvutil.o /usr/local/lib

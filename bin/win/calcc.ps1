param (
    [string]$d = ".",
    [string]$o = ".\bin",
    [string]$n = "out.exe",
    [switch]$h
)
if ($h) {
    "   CALCC - compiler for calc language"
    "   Early access version for windows"
    "   flags:"
    "       -d source code directory, default to '.'"
    "       -o output directory, default to '.\bin'"
    "       -n output executable name, default to 'out.exe'"
    exit
}
$ErrorActionPreference = "Stop"
mkdir "$o" -erroraction 'silentlycontinue'
& "$env:CALC_BIN\win\calccf.exe" -d $d -o $o\out.ll
if ($LastExitCode -ne 0) {
    "compile error"
    exit
}
$from = "$env:CALC_BIN\win\bdwgc\*.*"
$to = "$o"
Copy-Item -Path "$from" -Destination "$to"
$from = "$env:CALC_BIN\win\libuv\*.*"
Copy-Item -Path "$from" -Destination "$to"
clang $o\out.ll $o\libgc.dll.a $o\uv.lib $o\uvutil.a -static-libgcc -static-libstdc++ -lpthread  -o $o\$n
if ($LastExitCode -ne 0) {
    "compile error"
    exit
}
"success compiled to $o\$n"
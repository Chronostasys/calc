param (
    [string]$d = ".",
    [string]$o = ".\bin",
    [string]$n = "out.exe",
    [switch]$h,
    [switch]$ll
)
if ($h) {
    "   CALCC - compiler for calc language"
    "   Early access version for windows"
    "   flags:"
    "       -d source code directory, default to '.'"
    "       -o output directory, default to '.\bin'"
    "       -n output executable name, default to 'out.exe'"
    "       -h print help'"
    "       -ll emit llvm'"
    exit
}
$ErrorActionPreference = "Stop"
mkdir "$o" -erroraction 'silentlycontinue'
& "$env:CALC_BIN\win\calccf.exe" -d $d -o "$o\$n.ll"
if ($LastExitCode -ne 0) {
    "compile error"
    exit
}
if ($ll) {
    "llvm ir write to $o\$n.ll"
}
$from = "$env:CALC_BIN\win\bdwgc\*.*"
$to = "$o"
Copy-Item -Path "$from" -Destination "$to"
$from = "$env:CALC_BIN\win\libuv\*.*"
Copy-Item -Path "$from" -Destination "$to"
clang "$o\$n.ll" $o\libgc.dll.a $o\uv.lib $o\uvutil.a -static-libgcc -static-libstdc++ -lpthread  -o $o\$n
if ($LastExitCode -ne 0) {
    "compile error"
    Remove-Item -Path "$o" -Recurse
    exit
}

$arr = "*.dll","*.exe"
if ($ll) {
    $arr += "*.ll"
}
Remove-Item -Path "$o" -Exclude $arr  -Recurse
"success compiled to $o\$n"
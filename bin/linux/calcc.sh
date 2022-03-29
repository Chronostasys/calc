#!/bin/bash

helpFunction()
{
   echo -e "\tCalcc - calc languange compiler"
   echo "Usage: $0 -d . -o out -ll"
   echo -e "\t-d\tThe dir contains main module"
   echo -e "\t-o\tThe output executable name"
   echo -e "\t-ll\tEmit .ll file"
   exit 1 # Exit script after printing help
}

while getopts "d:o:ll:" opt
do
    case "$opt" in
      d ) ccdir="$OPTARG" ;;
      o ) outpath="$OPTARG" ;;
      ll ) llpath="$OPTARG" ;;
      ? ) helpFunction ;; # Print helpFunction in case parameter is non-existent
    esac
done

# # Print helpFunction in case parameters are empty
# if [ -z "$ccdir" ] || [ -z "$outpath" ] || [ -z "$llpath" ]
# then
#     echo "Some or all of the parameters are empty";
#     helpFunction
# fi

# Begin script in case all parameters are correct

set -e

if [ -z "$ccdir" ] 
then
    ccdir="."
fi

if [ -z "$outpath" ] 
then
    outpath="a.out"
fi

rmll="false"

if [ -z "$llpath" ] 
then
    llpath="out.ll"
    rmll="true"
fi

# echo "$ccdir"
# echo "$outpath"
# echo "$llpath"

calccf -d $ccdir -o $llpath
clang $llpath /usr/local/lib/uvutil.a /usr/local/lib/libuv.a /usr/local/lib/libgc.so  -ldl -static-libgcc -static-libstdc++ -lpthread  -o $outpath
if [ "$rmll" = "true" ]
then
    rm $llpath
fi
echo "Successfully compiled to $outpath"

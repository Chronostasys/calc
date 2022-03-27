#!/bin/bash

set -e

echo "***try to install build dependencies..."
sudo apt install -y clang autoconf libtool git
echo "***try to install bdwgc lib..."
ext="true"
if ! [ -d "bdwgc" ] 
then
    ext="false"
    git clone https://github.com/ivmai/bdwgc.git
fi
cd bdwgc
git checkout v8.2.0
if ! [ -d "libatomic_ops" ] 
then
    git clone https://github.com/ivmai/libatomic_ops.git
fi
libtoolize --automake --copy --debug --force
./autogen.sh
./configure
make install
cd ..
if [ $ext = "false" ]
then
    rm -rf bdwgc
fi
echo "***try to install libuv..."
ext="true"
if ! [ -d "libuv" ] 
then
    ext="false"
    git clone https://github.com/libuv/libuv.git
fi
cd libuv
git checkout v1.43.0
libtoolize --automake --copy --debug --force
./autogen.sh
./configure
make install
cd ..
if [ $ext = "false" ]
then
    rm -rf libuv
fi

rm ltmain.sh

echo "***successfully installed all dependencies"


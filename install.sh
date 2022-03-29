#!/bin/bash

set -e
GREEN="\e[32m"
ENDCOLOR="\e[0m"
RED="\e[31m"

echo -e "${GREEN}try to install build dependencies...${ENDCOLOR}"
sudo apt install -y clang autoconf libtool git
echo -e "${GREEN}try to install bdwgc lib${ENDCOLOR}"
if ! [ -d "bdwgc" ] 
then
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
sudo make install
cd ..
echo -e "${GREEN}try to install libuv...${ENDCOLOR}"
if ! [ -d "libuv" ] 
then
    git clone https://github.com/libuv/libuv.git
fi
cd libuv
git checkout v1.43.0
libtoolize --automake --copy --debug --force
./autogen.sh
./configure
sudo make install
cd ..
echo -e "${GREEN}try to install helpers${ENDCOLOR}"
cd libuv_helper
make install
if ! command -v go &> /dev/null
then
    echo -e "${RED}go command could not be found, please install golang first${ENDCOLOR}"
    echo -e "${RED}calcc install failed${ENDCOLOR}"
    exit 1
fi
echo -e "${GREEN}try to install calcc${ENDCOLOR}"
make install
echo -e "${GREEN}successfully installed calcc -- calc language compiler${ENDCOLOR}"


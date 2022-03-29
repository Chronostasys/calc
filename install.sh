#!/bin/bash

set -e
GREEN="\e[32m"
ENDCOLOR="\e[0m"
RED="\e[31m"

echo -e "${GREEN}try to install build dependencies...${ENDCOLOR}"
sudo apt-get update
sudo apt install -y clang autoconf libtool git
git submodule update --init --recursive
echo -e "${GREEN}try to install bdwgc lib${ENDCOLOR}"
cd bdwgc
git checkout v8.2.0
if ! [ -d "libatomic_ops" ] 
then
    git clone https://github.com/ivmai/libatomic_ops.git
fi
libtoolize --automake --copy --debug --force
cp ../ltmain.sh ltmain.sh
chmod +x ../ltmain.sh
chmod +x ltmain.sh
./autogen.sh
./configure
sudo make install
cd ..
echo -e "${GREEN}try to install libuv...${ENDCOLOR}"
cd libuv
git checkout v1.43.0
libtoolize --automake --copy --debug --force
cp ../ltmain.sh ltmain.sh
chmod +x ../ltmain.sh
chmod +x ltmain.sh
./autogen.sh
./configure
sudo make install
cd ..
echo -e "${GREEN}try to install helpers${ENDCOLOR}"
cd libuv_helper
make install
if ! command -v go &> /dev/null
then
    echo -e "${GREEN}go not found, try to install...${ENDCOLOR}"
    wget -q -O - https://raw.githubusercontent.com/canha/golang-tools-install-script/master/goinstall.sh | bash
    source ~/.bashrc
fi
echo -e "${GREEN}try to install calcc${ENDCOLOR}"
cd ..
make install
echo -e "${GREEN}successfully installed calcc -- calc language compiler${ENDCOLOR}"


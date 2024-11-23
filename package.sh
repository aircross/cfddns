#!/bin/sh
set -e


cfddns_version=$(cat version)
echo "build version: $cfddns_version"

# cross_compiles
make -f ./Makefile.cross-compiles

rm -rf ./release/packages
mkdir -p ./release/packages

os_all='linux windows darwin freebsd android'
arch_all='386 amd64 arm arm64 mips64 mips64le mips mipsle riscv64 loong64'
extra_all='_ hf'

cd ./release

for os in $os_all; do
    for arch in $arch_all; do
        for extra in $extra_all; do
            suffix="${os}_${arch}"
            if [ "x${extra}" != x"_" ]; then
                suffix="${os}_${arch}_${extra}"
            fi
            dir_name="cfddns_${cfddns_version}_${suffix}"
            cfddns_path="./packages/cfddns_${cfddns_version}_${suffix}"
            mkdir ${cfddns_path}
            if [ "x${os}" = x"windows" ]; then
                if [ ! -f "./cfddns_${os}_${arch}.exe" ]; then
                    continue
                fi
                mv ./cfddns_${os}_${arch}.exe ${cfddns_path}/cfddns.exe
            else
                if [ ! -f "./cfddns_${suffix}" ]; then
                    continue
                fi
                mv ./cfddns_${suffix} ${cfddns_path}/cfddns
            fi  
            cp -f ../conf.toml.example ${cfddns_path}


            # packages
            cd ./packages
            ls
            if [ "x${os}" = x"windows" ]; then
                zip -rq ${dir_name}.zip ${dir_name}
            else
                tar -zcf ${dir_name}.tar.gz ${dir_name}
            fi  
            cd ..
            rm -rf ${cfddns_path}
        done
    done
done
ls ./packages > ./packages/list.txt

cd -

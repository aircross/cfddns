#!/bin/sh
set -e


cfddns_version=$(cat version)
echo "build version: $cfddns_version"

# cross_compiles
make -f ./Makefile.cross-compiles

rm -rf ./release/packages
mkdir -p ./release/packages

# os_all='linux windows darwin freebsd android'
# arch_all='386 amd64 arm arm64 mips64 mips64le mips mipsle riscv64 loong64'
os_all='linux windows'
arch_all='386 amd64'
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
            # mkdir "./packages/${cfddns_version}"
            # cfddns_path="./packages/cfddns_${cfddns_version}_${suffix}"
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

            # if [ "x${os}" = x"windows" ]; then
            #     if [ ! -f "./cfddns_${os}_${arch}.exe" ]; then
            #         continue
            #     fi
            #     mkdir ${cfddns_path}
            #     mv ./frpc_${os}_${arch}.exe ${cfddns_path}/frpc.exe
            #     mv ./frps_${os}_${arch}.exe ${cfddns_path}/frps.exe
            # else
            #     if [ ! -f "./frpc_${suffix}" ]; then
            #         continue
            #     fi
            #     if [ ! -f "./frps_${suffix}" ]; then
            #         continue
            #     fi
            #     mkdir ${cfddns_path}
            #     mv ./frpc_${suffix} ${cfddns_path}/frpc
            #     mv ./frps_${suffix} ${cfddns_path}/frps
            # fi  
            # cp ../LICENSE ${cfddns_path}
            # cp -f ../conf/frpc.toml ${cfddns_path}
            # cp -f ../conf/frps.toml ${cfddns_path}
            ls


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

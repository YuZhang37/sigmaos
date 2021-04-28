#!/bin/bash

#
# Run from directory thas has "bin"
#

N=":1111"
if [ $# -eq 1 ]
then
    N=$1
fi

if [[ -z "${NAMED}" ]]; then
  export NAMED=$N
fi

./bin/memfsd 0 ":1111" &
./bin/nps3d &
./bin/npuxd &
./bin/locald ./ &

sleep 2
./mount.sh
mkdir -p /mnt/9p/fs   # make fake file system
mkdir -p /mnt/9p/kv
mkdir -p /mnt/9p/gg
mkdir -p /mnt/9p/memfsd-replicas

# Start a few memfs replicas
./bin/memfs-replica 1 ":30001" "192.168.0.36:30001" "192.168.0.36:30003" "192.168.0.36:30003" "192.168.0.36:30002" &
./bin/memfs-replica 2 ":30002" "192.168.0.36:30001" "192.168.0.36:30003" "192.168.0.36:30001" "192.168.0.36:30003" &
./bin/memfs-replica 3 ":30003" "192.168.0.36:30001" "192.168.0.36:30003" "192.168.0.36:30002" "192.168.0.36:30001" &

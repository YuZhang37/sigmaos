#!/bin/bash

if [ "$#" -ne 1 ]
then
  echo "Usage: $0 address"
  exit 1
fi

DIR=$(dirname $0)
source $DIR/env.sh

ssh -i $DIR/keys/cloudlab-sigmaos $LOGIN@$1 <<'ENDSSH'
# Turn off turbo boost.
echo 1 | sudo tee /sys/devices/system/cpu/intel_pstate/no_turbo

# Disable CPU frequency scaling.
# sudo cpupower frequency-set -g performance
np=$(nproc)
np=$((np-1))
for i in $(seq 0 $np) 
do
  echo "CPU frequency set core $i"
  sudo cpufreq-set -g performance -c $i
done
ENDSSH

echo "== TO LOGIN TO VM INSTANCE USE: =="
echo "ssh $1"
echo "============================="

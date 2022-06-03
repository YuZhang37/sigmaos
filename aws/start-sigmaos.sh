#!/bin/bash

usage() {
  echo "Usage: $0 [-n N] --vpc VPC" 1>&2
}

N_VM=""
VPC=""
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
  --vpc)
    shift
    VPC=$1
    shift
    ;;
  -n)
    shift
    N_VM=$1
    shift
    ;;
  -help)
    usage
    exit 0
    ;;
  *)
    echo "Error: unexpected argument '$1'"
    usage
    exit 1
    ;;
  esac
done

if [ -z "$VPC" ] || [ $# -gt 0 ]; then
    usage
    exit 1
fi

vms=`./lsvpc.py $VPC | grep -w VMInstance | cut -d " " -f 5`

vma=($vms)
MAIN="${vma[0]}"
NAMED="${vma[0]}:1111"
export NAMED="${NAMED}"

if ! [ -z "$N_VM" ]; then
  vms=${vma[@]:0:$N_VM}
fi

for vm in $vms; do
  ssh -i key-$VPC.pem ubuntu@$vm /bin/bash <<ENDSSH
  mkdir -p /tmp/ulambda/
  export NAMED="${NAMED}"
  if [ "${vm}" = "${MAIN}" ]; then 
    echo "START ${NAMED}"
    (cd ulambda; nohup ./start.sh > /tmp/start.out 2>&1 < /dev/null &)
  else
    echo "JOIN ${NAMED}"
    (cd ulambda; nohup bin/realm/noded . $vm > /tmp/noded.out 2>&1 < /dev/null &)
  fi
ENDSSH
done

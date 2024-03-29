#!/bin/bash
show-usage () {
  cat <<EOF
Usage: $(basename $0) [options]
Options:
  -h : show this message
  -t : release version tag. (default: snapshot)
  -g : Github OAuth token. (default : GITHUB_TOKEN env var)
  -i : ignore build errors
  -b : branch(default : master)
EOF
  exit 1
}

print-msg () { # log-msg level msg color
  local now=`date '+%Y/%m/%d %H:%M:%S'`
  local log=`printf "%-20s %-50s\n" "${now}" "${2}"`
  if [ ! -z "${3}" ]; then
    cRED=31; cGREEN=32; cYELLOW=33; cBLUE=34; cMAGENTA=35; cCYAN=36; cWHITE=37
    echo -e "\033[1;$(eval "echo \$c${3}")m${log}\033[0m"
  else
    echo ${1} | grep -E "[EW].*" > /dev/null 2>&1
    [ $? -eq 0 ] && echo -e "\033[1;31m${log}\033[0m" || echo "${log}"
  fi
  return 0
}

abort () {
  print-msg E "${1}"
  exit 1
}

handle-build-result () {
  if [ $1 -ne 0 ]; then
    if [ ${IGNORE_BUILD_ERROR} = 0 ]; then
      abort "Failed to build packages"
    else
      print-msg W "Failed to build some packages" YELLOW
    fi
  else
    print-msg W "All packages have been built successfully" CYAN
  fi
}

which greadlink >/dev/null 2>&1  && _readlink=greadlink || _readlink=readlink
SCRIPT_DIR=$(dirname $(${_readlink} -f $0))
cd "${SCRIPT_DIR}"

: ${GITHUB_TOKEN:=""}
: ${RELEASE_TAG:="snapshot"}
: ${IGNORE_BUILD_ERROR:=0}
: ${BRANCH:="master"}

while : ; do
  case "${1}" in
  -*)
    [[ "$1" =~ "h" ]] && show-usage
    if [[ "$1" =~ "i" ]]; then
      IGNORE_BUILD_ERROR=1
      shift 1
    elif [[ "$1" =~ "t" ]]; then
      if [[ -z "$2" || "$2" =~ "^-+" ]]; then
        echo "-t can not be empty";show-usage
      fi
      RELEASE_TAG="$2"
      shift 2
    elif [[ "$1" =~ "g" ]]; then
      if [[ -z "$2" || "$2" =~ "^-+" ]]; then
        echo "-g can not be empty";show-usage
      fi
      GITHUB_TOKEN="$2"
      shift 2
    elif [[ "$1" =~ "b" ]]; then
      if [[ -z "$2" || "$2" =~ "^-+" ]]; then
        echo "-b can not be empty";show-usage
      fi
      BRANCH="$2"
      shift 2
    fi
    ;;
  *)
    break
    ;;
  esac
done

_GO_VERSION=`go version`
[ $? -ne 0 ] && abort "'go' command not found on PATH"
if ! which gox >/dev/null 2>&1 ; then
  print-msg I "'gox' command not found on PATH."
  print-msg I "Installing gox..."
  go install github.com/mitchellh/gox@latest
  [ $? -ne 0 ] && abort "Failed to install gox"
fi
if ! which ghr >/dev/null 2>&1 ; then
  print-msg I "'ghr' command not found on PATH."
  print-msg I "Installing ghr..."
  go install github.com/tcnksm/ghr@latest
  [ $? -ne 0 ] && abort "Failed to install ghr"
fi

CPU_NUM=$(python3 -c 'import multiprocessing; print(multiprocessing.cpu_count())')
print-msg I "num of cpus: ${CPU_NUM}"

GOX_OSARCHS="darwin/amd64 darwin/arm64 linux/386 linux/amd64 linux/arm64 linux/armv6 windows/386 windows/amd64 windows/arm64"

print-msg I "${_GO_VERSION}"
print-msg I "tag: ${RELEASE_TAG}"
_OLD_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "${_OLD_BRANCH}" != "${BRANCH}" ]; then
  print-msg I "git checkout ${BRANCH}"
  git checkout ${BRANCH}
  [ $? -ne 0 ] && abort "Failed to checkout ${BRANCH}"
fi

if [ "${RELEASE_TAG}" != "snapshot" ]; then
  print-msg I git checkout refs/tags/${RELEASE_TAG}
  git checkout refs/tags/${RELEASE_TAG}
  [ $? -ne 0 ] && abort "Failed to checkout the tag ${RELEASE_TAG}"
fi

rm -rf "${SCRIPT_DIR}/packages"

print-msg I "gox -osarch "${GOX_OSARCHS}" -output=${SCRIPT_DIR}/packages/{{.Dir}}_${RELEASE_TAG}_{{.OS}}_{{.Arch}}" -ldflags="-s"
env CGO_ENABLED=0 gox -osarch "${GOX_OSARCHS}" -output="${SCRIPT_DIR}/packages/{{.Dir}}_${RELEASE_TAG}_{{.OS}}_{{.Arch}}" -ldflags="-s"
handle-build-result $?

_NUM_THREADS=${CPU_NUM}
if [ -z "${CPU_NUM}" -o ${CPU_NUM} -lt 4 ]; then
  _NUM_THREADS=4
fi

print-msg I "ghr --parallel=${_NUM_THREADS} --delete --token=**** ${RELEASE_TAG} packages"
ghr --parallel=${_NUM_THREADS} --delete --token=${GITHUB_TOKEN} ${RELEASE_TAG} packages
[ $? -ne 0 ] && abort "Failed to upload some packages"
print-msg W "All packages have been uploaded successfully" CYAN

if [ "${_OLD_BRANCH}" != "${BRANCH}" ]; then
  print-msg I "git checkout ${_OLD_BRANCH}"
  git checkout ${_OLD_BRANCH}
  [ $? -ne 0 ] && abort "Failed to checkout ${_OLD_BRANCH}"
fi

rm -rf "${SCRIPT_DIR}/packages"
print-msg I "OK" CYAN

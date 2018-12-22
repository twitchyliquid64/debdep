#!/bin/bash
set -e

SCRIPT_BASE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
BUILD_DIR=${1%/}

if [ -f "${BUILD_DIR}" ]; then
	echo "Cannot build into '${BUILD_DIR}': it is a file."
	exit 1
fi

if [ -d "${BUILD_DIR}" ]; then
  rm -rfv ${BUILD_DIR}/*
fi

mkdir -pv ${BUILD_DIR}

go build -o "${BUILD_DIR}/usr/bin/debdep" github.com/twitchyliquid64/debdep/debdep

mkdir -pv "${BUILD_DIR}/DEBIAN"
cp -rv ${SCRIPT_BASE_DIR}/DEBIAN/* "${BUILD_DIR}/DEBIAN"


mkdir -pv "${BUILD_DIR}/usr/share/man/man1"
cp "${SCRIPT_BASE_DIR}/debdep.1" "${BUILD_DIR}/usr/share/man/man1/debdep.1"
gzip "${BUILD_DIR}/usr/share/man/man1/debdep.1"

dpkg-deb --build "${BUILD_DIR}" ./

#!/bin/sh
MAINDIR=$PWD
HASERROR=false
TESTDIRS=(. cli pipe plugin)
COLOR_CLEAR="\x1b[0m"
COLOR_OK="\x1b[32;01m"
COLOR_ERROR="\x1b[31;01m"
echo -e "==> Run tests"
for name in ${TESTDIRS[@]}; do
    path=${MAINDIR}/${name}
    [ -d "${path}" ] || continue # if not a directory, skip
    dirname="$(basename "${path}")"
    cd ${path}
    echo -e "Test: ${dirname}"
    go test .
    if [ $? == 0 ]; then
		echo -e " - ${COLOR_OK}OK${COLOR_CLEAR}"
	else
		echo -e " - ${COLOR_ERROR}FAIL${COLOR_CLEAR}"
		HASERROR=true
	fi
    
done
echo -en "==> Tests"
if [ "$HASERROR" = true ] ; then
	echo -e " - ${COLOR_ERROR}FAIL${COLOR_CLEAR}"
	echo -e "There were testing errors"
	exit 1
fi
echo -e " - ${COLOR_OK}OK${COLOR_CLEAR}"
exit 0
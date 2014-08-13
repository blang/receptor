#!/bin/sh
LOGFILE=$PWD/build.log
HASERROR=false

COLOR_CLEAR="\x1b[0m"
COLOR_OK="\x1b[32;01m"
COLOR_ERROR="\x1b[31;01m"

echo -en "==> Building receptor"

mkdir -p -- "./bin" >> ${LOGFILE} 2>&1
go get ./cli >> ${LOGFILE} 2>&1
go build -o "./bin/receptor" ./cli >> ${LOGFILE} 2>&1

if [ $? == 0 ]; then
	echo -e " - ${COLOR_OK}OK${COLOR_CLEAR}"
else
	echo -e " - ${COLOR_ERROR}FAIL${COLOR_CLEAR}"
	HASERROR=true
fi

if [ "$HASERROR" = true ] ; then
	echo -e "There are building errors, see ${LOGFILE}"
	exit 1
fi
exit 0
#!/bin/sh
PLUGINDIR=$PWD/bin/plugins
PLUGINSRCDIR=$PWD/plugins
LOGFILE=$PWD/build.log
HASERROR=false

COLOR_CLEAR="\x1b[0m"
COLOR_OK="\x1b[32;01m"
COLOR_ERROR="\x1b[31;01m"

mkdir -p ${PLUGINDIR}

echo -e "==> Building watcher plugins"
for path in ${PLUGINSRCDIR}/watcher/*; do
    [ -d "${path}" ] || continue # if not a directory, skip
    dirname="$(basename "${path}")"
    cd ${path}
    BINNAME=receptor-watcher-${dirname}
    echo -en "${BINNAME}"
    go get ./... >> ${LOGFILE} 2>&1
    go build -o ${BINNAME} . >> ${LOGFILE} 2>&1 && mv ${BINNAME} ${PLUGINDIR}/
    if [ $? == 0 ]; then
		echo -e " - ${COLOR_OK}OK${COLOR_CLEAR}"
	else
		echo -e " - ${COLOR_ERROR}FAIL${COLOR_CLEAR}"
		HASERROR=true
	fi
    
done

echo -e "==> Building reactor plugins"
for path in ${PLUGINSRCDIR}/reactor/*; do
    [ -d "${path}" ] || continue # if not a directory, skip
    dirname="$(basename "${path}")"
    cd ${path}
    BINNAME=receptor-reactor-${dirname}
    echo -en "${BINNAME}"
    go get ./... >> ${LOGFILE} 2>&1 
    go build -o ${BINNAME} . >> ${LOGFILE} 2>&1 && mv ${BINNAME} ${PLUGINDIR}/
    if [ $? == 0 ]; then
		echo -e " - ${COLOR_OK}OK${COLOR_CLEAR}"
	else
		echo -e " - ${COLOR_ERROR}FAIL${COLOR_CLEAR}"
		HASERROR=true
	fi
done

if [ "$HASERROR" = true ] ; then
	echo -e "There are building errors, see ${LOGFILE}"
	exit 1
fi
exit 0
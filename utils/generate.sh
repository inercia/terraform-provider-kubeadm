#!/bin/sh
# a simple script for generating a golang file where we store
# the contents of the files in a constant
#

log()   { echo ">>> $1" ; }
warn()  { log "WARNING: $1" ; }
abort() { log "FATAL: $1" ; exit 1 ; }

IN_FILES=
OUT_FILE="generated.go"
OUT_PACKAGE="main"
OUT_VAR="text"

while [ $# -gt 0 ] ; do
  case $1 in
    --out-file)
      OUT_FILE=$2
      shift
      ;;
    --out-package)
      OUT_PACKAGE=$2
      shift
      ;;
    --out-var)
      OUT_VAR=$2
      shift
      ;;
    *)
      IN_FILES="$IN_FILES $1"
      ;;
  esac
  shift
done

[ -z "$IN_FILES"    ] && abort "no input files provided"
[ -z "$OUT_FILE"    ] && abort "no output files provided"
[ -z "$OUT_PACKAGE" ] && abort "no output package provided"
[ -z "$OUT_VAR"     ] && abort "no output variable provided"

echo    "package $OUT_PACKAGE"            > $OUT_FILE
echo                                     >> $OUT_FILE
echo   "// file automatically generated" >> $OUT_FILE
echo   "// DO NOT MODIFY"                >> $OUT_FILE
echo                                     >> $OUT_FILE
echo -n "const $OUT_VAR=\`"              >> $OUT_FILE
cat     $IN_FILES | sed -e 's|`|\\`|g'   >> $OUT_FILE
echo    "\`"                             >> $OUT_FILE

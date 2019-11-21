#!/bin/sh

TFCHEK_PORT=$PORT
export TFCHEK_PORT
if [ -r "/configs/tfChek" ]; then
  source /configs/tfChek
  echo "Environment variables has been sourced" 1>&2
else
  echo "WARNING!!! Environment variables has not been sourced" 1>&2
fi
echo "Launching $*" 1>&2
./tfChek "$@"
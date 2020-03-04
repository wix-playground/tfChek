#!/bin/sh

TFCHEK_PORT=$PORT
export TFCHEK_PORT
echo -e "\033[0;32mConfigured tfChek to listen to the port\033[0;35m $TFCHEK_PORT\033[0m"
echo "Launching $*" 1>&2
./tfChek "$@"
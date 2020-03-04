#!/bin/bash

TFCHEK_PORT=$PORT
export TFCHEK_PORT
echo -e "\033[0;32mConfigured tfChek to listen to the port\033[0;35m $TFCHEK_PORT\033[0m"
#Preparing keys
mkdir ${HOME}/.ssh && chmod 700 ${HOME}/.ssh
cat /configs/id_rsa | sed 's~ RSA PRIVATE KEY~RSAPRIVATEKEY~g'| sed 's~[ ]~\n~g' | sed 's~RSAPRIVATEKEY~ RSA PRIVATE KEY~g' >${HOME}/.ssh/id_rsa
chmod 400 ${HOME}/.ssh/id_rsa
chown $USER ${HOME}/.ssh/id_rsa
echo -e "\033[0;32mSSH keys are ready\033[0m" && ( rm -f /configs/id_rsa || echo -e "\033[0;31mCannot remove id_rsa from configs directory\033[0m" )

echo "Launching $*" 1>&2
./tfChek "$@"

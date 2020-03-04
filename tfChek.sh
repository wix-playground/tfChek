#!/usr/bin/env bash

TFCHEK_PORT=$PORT
export TFCHEK_PORT
echo -e "\033[0;32mConfigured tfChek to listen to the port\033[0;35m $TFCHEK_PORT\033[0m"
#Preparing keys
mkdir ~/.ssh && chmod 700 ~/.ssh
cat /configs/id_rsa | sed 's~ RSA PRIVATE KEY~RSAPRIVATEKEY~g'| sed 's~[ ]~\n~g' | sed 's~RSAPRIVATEKEY~ RSA PRIVATE KEY~g' > ~/.ssh/id_rsa
chmod 400 ~/.ssh/id_rsa
chown $(whoami) ~/.ssh/id_rsa
echo -e "\033[0;32mSSH keys are ready\033[0m"

echo "Launching $*" 1>&2
./tfChek "$@"

#!/usr/bin/env bash

TFCHEK_PORT=$PORT
export TFCHEK_PORT
echo -e "\033[0;32mOK\033[0m Configured tfChek to listen to the port\033[0;35m $TFCHEK_PORT\033[0m"
#TODO: improve this ugly workaround
#Preparing keys
mkdir ~/.ssh && chmod 700 ~/.ssh
cat /configs/id_rsa | sed 's~ RSA PRIVATE KEY~RSAPRIVATEKEY~g'| sed 's~[ ]~\n~g' | sed 's~RSAPRIVATEKEY~ RSA PRIVATE KEY~g' > ~/.ssh/id_rsa
chmod 400 ~/.ssh/id_rsa
chown $(whoami) ~/.ssh/id_rsa
echo -e "\033[0;32mOK\033[0m SSH keys are ready\033[0m"
eval "$(ssh-agent)" && echo -e "\033[0;32mOK\033[0m SSH agent has been started\033[0m" || echo -e "\033[0;31mERROR\033[0m Cannot start ssh agent"

echo "Launching $*" 1>&2
./tfChek "$@"

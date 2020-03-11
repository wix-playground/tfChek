#!/usr/bin/env bash

TFCHEK_PORT=$PORT
export TFCHEK_PORT
echo -e "\033[0;32mOK\033[0m Configured tfChek to listen to the port\033[0;35m $TFCHEK_PORT\033[0m"
#TODO: improve this ugly workaround
#Preparing keys
cat /configs/id_rsa | sed 's~ RSA PRIVATE KEY~RSAPRIVATEKEY~g'| sed 's~[ ]~\n~g' | sed 's~RSAPRIVATEKEY~ RSA PRIVATE KEY~g' > ~/.ssh/id_rsa
chmod 400 ~/.ssh/id_rsa
chown $(whoami) ~/.ssh/id_rsa
echo -e "\033[0;32mOK\033[0m SSH keys are present\033[0m"
eval "$(ssh-agent)" && echo -e "\033[0;32mOK\033[0m SSH agent has been started\033[0m" || echo -e "\033[0;31mERROR\033[0m Cannot start ssh agent"
ssh-add && echo -e "\033[0;32mOK\033[0m SSH key has been added to the SSH Agent\033[0m" || echo -e "\033[0;31mERROR\033[0m Cannot add SSH key to the agent"

#Prepare Chef keys
cat /configs/validation | sed 's~ RSA PRIVATE KEY~RSAPRIVATEKEY~g'| sed 's~[ ]~\n~g' | sed 's~RSAPRIVATEKEY~ RSA PRIVATE KEY~g' > ~/.chef/validation.pem
cat /configs/databag | sed 's~ RSA PRIVATE KEY~RSAPRIVATEKEY~g'| sed 's~[ ]~\n~g' | sed 's~RSAPRIVATEKEY~ RSA PRIVATE KEY~g' > ~/.chef/data_bag_secret
chown $(whoami) ~/.chef/validation.pem ~/.chef/data_bag_secret
chmod 400 ~/.chef/validation.pem ~/.chef/data_bag_secret
echo -e "\033[0;32mOK\033[0m Chef keys are present\033[0m"

#As a temporary measure add start wixress key and certificates
cp /configs/swp_key_t2020 ~/luggage/star_wixpress_com_until_2020.key
chmod 400 ~/luggage/star_wixpress_com_until_2020.key
[ -r ~/luggage/star_wixpress_com_until_2020.key ] &&  echo -e "\033[0;32mOK\033[0m star wixpress certificate key is present \033[0m" || echo -e "\033[0;31mERROR\033[0m star wixpress certificate key is missing"

#Prepare AWS credentials
cp /configs/aws_credentials ~/.aws/credentials && chmod 400 ~/.aws/credentials

#Source profile
source /configs/profile && echo -e "\033[0;32mOK\033[0m Bash profile has been successfully sourced" || echo -e "\033[0;31mERROR\033[0m Failed to source bash profile"

echo "Launching $*" 1>&2
./tfChek "$@"

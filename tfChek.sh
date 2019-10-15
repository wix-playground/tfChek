#!/usr/bin/env bash

TFCHEK_PORT=$PORT
export TFCHEK_PORT
if [ -r "/configs/tfChek" ]; then
  source /configs/tfChek
fi
./tfChek
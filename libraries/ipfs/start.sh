#!/usr/bin/env bash
ipfs init
ipfs config Addresses.Gateway /ip4/${IP_ADDR:=0.0.0.0}/${GATE_PROTO:=tcp}/${GATE_PORT:=8080}
ipfs config Addresses.API /ip4/${IP_ADDR:=0.0.0.0}/${API_PROTO:=tcp}/${API_PORT:=5001}
ipfs daemon -writable


#!/bin/sh

echo "Hello There! I'm your friendly blockchain container."
echo ""
echo "My GENESIS.JSON which you sent me is:"
echo $GENESIS
echo ""
echo ""
echo "I'm Starting to build the Chain. And setting local_vars."
echo $GENESIS > /root/genesis.json
key_session="$(strings /dev/urandom | grep -o '[[:alnum:]]' | head -n 10 | tr -d '\n' ; echo)"

epm new --name ${CHAIN_NAME:=this_chain} --checkout --genesis /root/genesis.json
epm config key_session:${KEY_SESSION:=key_session} \
  local_host:${LOCAL_HOST:=0.0.0.0} \
  local_port:${LOCAL_PORT:=15254} \
  max_peers:${MAX_PEERS:=10}

echo ""
echo ""
echo "RPC Check."
if [ "$RPC" = true ]
then
  epm config serve_rpc:true rpc_host:$RPC_HOST rpc_port:${RPC_PORT:=15255}
fi

echo ""
echo ""
echo "Connect Check."
if [ "$CONNECT" = true ]
then
  epm config remote_host:$REMOTE_HOST remote_port:$REMOTE_PORT use_seed:true
fi

echo ""
echo ""
echo "SOLO Check."
if [ "$SOLO" = true ]
then
  epm config listen:false
fi

echo ""
echo ""
echo "Mining Check."
if [ "$MINING" = true ]
then
  epm config mining:true
fi

echo ""
echo ""
echo "Starting up! (Wheeeeeee says the marmot)"
epm --log ${LOG_LEVEL:=5} run
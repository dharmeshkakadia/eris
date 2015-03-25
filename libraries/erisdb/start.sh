#!/bin/sh

echo ""
echo ""
echo "Hello There! I'm your friendly blockchain container."
key_session="$(strings /dev/urandom | grep -o '[[:alnum:]]' | head -n 10 | tr -d '\n' ; echo)"

echo ""
echo ""
echo "Checking if Master"
if [ "$MASTER" = "true" ]
then
  echo ""
  echo ""
  echo "I'm a master node so I will build a chain now."
  echo ""
  echo ""
  if [ -z "$GENESIS" ]
  then
    echo "No GENESIS variable was given will try volume approach."
    if [ -f /home/$user/genesis/genesis.json ]
    then
      echo "The GENESIS.JSON which you sent me is (the first 50 lines only...):"
      head -n 50 /home/$user/genesis/genesis.json
    else
      echo "No GENESIS.JSON given, using default:"
    fi
    echo ""
    echo ""
  else
    echo "The GENESIS.JSON which you sent me is:"
    echo $GENESIS
    echo $GENESIS > /home/$user/genesis/genesis.json
    echo ""
    echo ""
  fi
  if [ -f /home/$user/genesis/genesis.json ]
  then
    epm --log ${LOG_LEVEL:=3} new --name ${CHAIN_NAME:=this_chain} --checkout --genesis /home/$user/genesis/genesis.json
  else
    epm --log ${LOG_LEVEL:=3} new --name ${CHAIN_NAME:=this_chain} --checkout --no-edit
  fi
  epm config key_session:$key_session
  epm run & sleep 3 && kill $(epm plop pid)
  echo "The chain has been built and checked out."
else
  echo "I'm not a master."
fi

echo ""
echo ""
echo "Checking if Fetcher"
if [ "$FETCH" = "true" ]
then
  echo "I'm supposed to fetch so I will grab the chain from $REMOTE_HOST:$REMOTE_FETCH_PORT."
  echo ""
  epm --log ${LOG_LEVEL:=3} fetch --checkout --name ${CHAIN_NAME:=this_chain} $REMOTE_HOST:$REMOTE_FETCH_PORT
  echo ""
  echo "Catching up the chain from $REMOTE_HOST:$REMOTE_PORT. This will take a few seconds."
  echo ""
  epm config key_session:$key_session remote_host:$REMOTE_HOST remote_port:$REMOTE_PORT use_seed:true
  epm --log ${LOG_LEVEL:=3} run & sleep 30 && kill $(epm plop pid)
  echo "The chain has been fetched and checked out."
else
  echo "I'm not a fetcher."
fi

echo ""
echo ""
echo "Setting Defaults"
epm config key_session:$key_session \
  local_host:${LOCAL_HOST:=0.0.0.0} \
  local_port:${LOCAL_PORT:=15254} \
  max_peers:${MAX_PEERS:=10}

echo ""
echo ""
echo "Setting the Key File"
if [ -z "$KEY_FILE" ]
then
  echo "No key file given."
else
  echo "Using the given key file."
  epm keys use ${KEY_FILE}
fi

echo ""
echo ""
echo "RPC Check."
if [ "$RPC" = "true" ]
then
  epm config serve_rpc:true rpc_host:${RPC_HOST:=0.0.0.0} rpc_port:${RPC_PORT:=15255}
fi

echo ""
echo ""
echo "Connect Check."
if [ "$CONNECT" = "true" ]
then
  epm config remote_host:$REMOTE_HOST remote_port:$REMOTE_PORT use_seed:true
fi

echo ""
echo ""
echo "SOLO Check."
if [ "$SOLO" = "true" ]
then
  epm config listen:false
fi

echo ""
echo ""
echo "Mining Check."
if [ "$MINING" = "true" ]
then
  epm config mining:true
fi

echo ""
echo ""
echo "Fetch Serve Check."
if [ "$SERVE_GBLOCK" = "true" ]
then
  epm config fetch_port:${FETCH_PORT:=50505}
fi

echo ""
echo ""
echo "My CHAINID is ... ->"
epm plop chainid
CHAINID=$(epm plop chainid)

echo ""
echo ""
echo "My Public Address is ... ->"
epm plop addr

echo ""
echo ""
echo "Starting up! (Wheeeeeee says the marmot)"
exec epm --log ${LOG_LEVEL:=3} run
#!/bin/sh

cd ~/.eris/dapps/helloworld

echo ""
echo ""
echo "My environment good sir:"
# this is here as I find it helpful for debugging, feel free to remove
printenv

echo ""
echo ""
echo "Am fetching the genesis block and catching up the chain."
echo ""
# first we need to fetch from the blockchain deployed via the
#   erisdb container (that is a raw, no contracts, no genDoug
#   chain).
epm fetch --checkout --name helloworld helloworldmaster:15256
# the sleep is here to allow epm run to connect to the peer
#   and pull the chain.
epm --log ${LOG_LEVEL:=3} run & sleep 2 && kill $(epm plop pid)

# Ensure unique key_sessions && set default local port and local_host (binding these
#   to hostname via env vars so that the host which the peer broadcasts will operate
#   properly and the peers can find each other as linked containers).
key_session="$(strings /dev/urandom | grep -o '[[:alnum:]]' | head -n 10 | tr -d '\n' ; echo)"
epm config key_session:${KEY_SESSION:=$key_session}
epm config local_host:0.0.0.0 local_port:15254

# when connecting to a remote chain these are the necessary minimums
remote_host="helloworldmaster"
remote_port="15254"
epm config remote_host:$remote_host remote_port:$remote_port use_seed:true

# now its time to deploy the contracts on
#   the checked out chain.
cd contracts && epm --log 4 deploy && cd ..
# another small catchup command to make the chain plays nice
#   post deployment
epm --log ${LOG_LEVEL:=3} run & sleep 2 && kill $(epm plop pid)

# Capture the primary variables
BLOCKCHAIN_ID=$(epm plop chainid)
if [ -z $ROOT_CONTRACT ] || [ $ROOT_CONTRACT == "" ]
then
  ROOT_CONTRACT=$(epm plop vars | cut -d : -f 2)
fi

echo ""
echo ""
echo "Configuring package.json with BLOCKCHAIN_ID ($BLOCKCHAIN_ID) and "
echo "ROOT_CONTRACT ($ROOT_CONTRACT) and "
echo "PEER_SERVER ($remote_host:$remote_port)"
mv package.json /tmp/
jq '.module_dependencies[0].data |= . * {peer_server_address: "'$remote_host:$remote_port'", blockchain_id: "'$BLOCKCHAIN_ID'", root_contract: "'$ROOT_CONTRACT'"}' /tmp/package.json \
    > ~/.eris/dapps/helloworld/package.json

echo ""
echo ""
echo "My package.json now looks like this."
# this is here for debugging, feel free to remove
cat ~/.eris/dapps/helloworld/package.json

echo ""
echo ""
echo "My chain config now looks like this."
epm plop config

# put the helloworld DApp in focus
echo ""
echo ""
echo "Starting up! (Wheeeeeee says the marmot)"
sleep 5 && curl http://localhost:3000/admin/switch/helloworld &
decerver

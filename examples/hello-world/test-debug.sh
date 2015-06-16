#!/bin/sh

# start the background containers
docker-compose up --no-recreate compilers ipfs helloworldmaster &
sleep 5 # give the master a bit of time to get everything sorted

# start the writer
docker-compose up --no-recreate helloworldwrite &
sleep 30 # give the writer time to catch up with master and deploy contracts

# grab the root contract from the writer
helloworldwrite=$(docker-compose ps -q helloworldwrite)
export ROOT_CONTRACT=$(docker logs $helloworldwrite | grep ROOT_CONTRACT | tail -n 1 | cut -c 16-57)

# helpful for debugging
echo ""
echo ""
echo "Hello World Writer's DOUG Contract is at:"
echo $ROOT_CONTRACT
echo ""

# start the reader
docker-compose up --no-recreate helloworldread &

docker-compose up --no-recreate seleniumnode &

docker-compose run helloworldtest
docker-compose kill
docker-compose rm --force
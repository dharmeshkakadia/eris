#!/bin/sh

# start the background containers
docker build -t 2gather .
docker run -d --name=compilers eris/compilers:latest
docker run -d -p 4001:4001 -p 8080:8080 --name=ipfs eris/ipfs:latest
docker run -d -e MASTER=true -e SERVE_GBLOCK=true -e FETCH_PORT=15256 \
  --name=helloworldmaster --expose=15254 --expose=15256 eris/erisdb:latest
sleep 5 # give the master a bit of time to get everything sorted

# start the writer
docker run -d -p 3000:3000 -e CONTAINERS=true --expose=15254 --expose=3000 \
  --link=compilers:compilers --link=ipfs:ipfs --link=helloworldmaster:helloworldmaster \
  --name=helloworldwrite 2gather
sleep 30 # give the writer time to catch up with master and deploy contracts

# grab the root contract from the writer
export ROOT_CONTRACT=$(docker logs helloworldwrite | grep ROOT_CONTRACT | tail -n 1 | cut -c 16-57)

# helpful for debugging
echo ""
echo ""
echo "Hello World Writer's DOUG Contract is at:"
echo $ROOT_CONTRACT
echo ""

# start the reader
docker run -d -p 3001:3000 -e CONTAINERS=true --expose=15254 --expose=3000 \
  --link=compilers:compilers --link=ipfs:ipfs --link=helloworldmaster:helloworldmaster \
  -e ROOT_CONTRACT=$ROOT_CONTRACT --link=helloworldwrite:helloworldwrite --name=helloworldread 2gather

docker run -d --expose=4444 --name=seleniumhub selenium/hub:latest
docker run -d --link=seleniumhub:hub --link=helloworldwrite:helloworldwrite \
  --link=helloworldread:helloworldread -p 5900:5900 --name=seleniumchrome \
  selenium/node-chrome:latest

cd test
docker build -t hw_test .
docker run --name helloworldtester --link=seleniumhub:selenium hw_test
cd ..

# shutdown
docker kill compilers && docker rm compilers
docker kill ipfs && docker rm ipfs
docker kill helloworldmaster && docker rm helloworldmaster
docker kill helloworldwrite && docker rm helloworldwrite
docker kill helloworldread && docker rm helloworldread
docker kill seleniumhub && docker rm seleniumhub
docker kill seleniumchrome && docker rm seleniumchrome
docker kill helloworldtester && docker rm helloworldtester

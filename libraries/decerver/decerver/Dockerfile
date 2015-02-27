# Dependencies

## Make sure your machine has >= 1 GB of RAM.

## Make sure go version >= 1.3.3 is installed and set up
FROM golang:1.4
MAINTAINER Eris Industries <contact@erisindustries.com>

### The base image kills /var/lib/apt/lists/*.
RUN apt-get update
RUN apt-get install -y \
  libgmp3-dev

## Copy In the Good Stuff -- This depends on `go build` rather than `go install`
COPY cmd/decerver/decerver $GOPATH/bin/decerver

## How Does It Run?
EXPOSE 3000 3005 4001 30303 30304
CMD ["decerver"]
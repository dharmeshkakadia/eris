# Dependencies

## Make sure your machine has >= 1 GB of RAM.

FROM golang:1.4
MAINTAINER Eris Industries <support@erisindustries.com>

# Install tools for editing configuration files.
RUN apt-get update && apt-get install -qy \
  jq \
  nano \
  vim \
  libgmp3-dev

RUN mkdir --parents $GOPATH/src/github.com/eris-ltd
WORKDIR $GOPATH/src/github.com/eris-ltd

# Install EPM.
COPY epm-go $GOPATH/src/github.com/eris-ltd/epm-go
RUN cd epm-go/cmd/epm; go get .

# Install Decerver.
COPY decerver $GOPATH/src/github.com/eris-ltd/decerver
RUN cd decerver/cmd/decerver; go get .

# Configure

ENV user eris
RUN groupadd --system $user && useradd --system --create-home --gid $user $user

## Listen to clients outside the container.
COPY config /home/$user/.decerver/

## Use the local compiler.
COPY config.json /home/$user/.decerver/languages/

RUN chown --recursive $user /home/$user/.decerver

# Expose ports.
# https://github.com/eris-ltd/decerver/wiki/Ports

## HTTP receiver
EXPOSE 3000 3005

## IPFS
EXPOSE 4001 4002/udp

## Thelonious
EXPOSE 30303 30304

# Hard-code 'eris' instead of using $user to support Docker versions < 1.3.
USER eris
CMD ["decerver"]

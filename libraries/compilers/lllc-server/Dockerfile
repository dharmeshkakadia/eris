FROM ubuntu:14.04
MAINTAINER Eris Industries <support@erisindustries.com>

# Ethereum-CPP Dependencies
ENV DEBIAN_FRONTEND noninteractive
RUN apt-get install --no-install-recommends -qy software-properties-common && \
  add-apt-repository ppa:ethereum/ethereum && \
  apt-get update && apt-get install --no-install-recommends -qy \
  automake build-essential cmake g++-4.8 git curl gcc make scons yasm \
  wget unzip libtool libgmp-dev libleveldb-dev libc6-dev libboost-all-dev \
  libminiupnpc-dev libreadline-dev libjsoncpp-dev libargtable2-dev \
  libncurses5-dev libcryptopp-dev libjson-rpc-cpp-dev libcurl4-openssl-dev \
  && rm -rf /var/lib/apt/lists/*

# Go
ENV GOLANG_VERSION 1.4.1

RUN curl -sSL https://golang.org/dl/go$GOLANG_VERSION.src.tar.gz \
    | tar -v -C /usr/src -xz

RUN cd /usr/src/go/src && ./make.bash --no-clean 2>&1

ENV PATH /usr/src/go/bin:$PATH

RUN mkdir -p /go/src
ENV GOPATH /go
ENV PATH /go/bin:$PATH
WORKDIR /go

# Install LLL (eris LLL, which includes a few extra opcodes)
ENV repository eris-cpp
RUN mkdir /usr/local/src/eris-cpp
WORKDIR /usr/local/src/eris-cpp

RUN curl --location https://github.com/eris-ltd/eris-cpp/archive/master.tar.gz \
 | tar --extract --gzip --strip-components=1

WORKDIR build
RUN bash instructions

# LLLC-server
RUN mkdir --parents $GOPATH/src/github.com/eris-ltd/lllc-server
COPY . $GOPATH/src/github.com/eris-ltd/lllc-server/
RUN cd $GOPATH/src/github.com/eris-ltd/lllc-server/cmd/lllc-server && go get .

# Add Eris user
RUN groupadd --system eris && useradd --system --create-home --gid eris eris

## Point to the compiler location.
COPY tests/config.json /home/eris/.decerver/languages/
RUN chown --recursive eris /home/eris

USER eris
WORKDIR /home/eris

EXPOSE 9099
CMD ["lllc-server"]

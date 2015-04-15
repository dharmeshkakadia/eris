FROM ubuntu:14.04
MAINTAINER Eris Industries <support@erisindustries.com>

# Install Golang
ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update && apt-get install -qy \
  ca-certificates \
  curl \
  gcc \
  git \
  libc6-dev
ENV GOLANG_VERSION 1.4.2
RUN curl -sSL https://golang.org/dl/go$GOLANG_VERSION.src.tar.gz \
  | tar -v -C /usr/src -xz
RUN cd /usr/src/go/src && ./make.bash --no-clean 2>&1
ENV PATH /usr/src/go/bin:$PATH
RUN mkdir -p /go/src /go/bin && chmod -R 777 /go
ENV GOPATH /go
ENV PATH /go/bin:$PATH
WORKDIR /go

# Install the building dependencies
RUN apt-get install -qy \
  automake \
  build-essential \
  cmake \
  g++-4.8 \
  libargtable2-dev \
  libboost-all-dev \
  libcurl4-openssl-dev \
  libgmp-dev \
  libjsoncpp-dev \
  libleveldb-dev \
  libminiupnpc-dev \
  libncurses5-dev \
  libreadline-dev \
  libtool \
  make \
  scons \
  software-properties-common \
  wget \
  yasm \
  unzip

# Install serpent
ENV repository serpent
RUN git clone https://github.com/ethereum/serpent /usr/local/src/$repository
WORKDIR /usr/local/src/$repository
RUN git checkout develop
RUN make && make install

# Solc
RUN add-apt-repository -y ppa:ethereum/ethereum && \
  add-apt-repository -y ppa:ethereum/ethereum-qt && \
  add-apt-repository -y ppa:ethereum/ethereum-dev && \
  apt-get update && apt-get install -qy \
  libcryptopp-dev \
  libjson-rpc-cpp-dev \
  lllc \
  solc \
  && rm -rf /var/lib/apt/lists/*

# Install Eris LLL, which includes a few extra opcodes
ENV repository eris-cpp
RUN mkdir /usr/local/src/$repository
WORKDIR /usr/local/src/$repository
RUN curl --location https://github.com/eris-ltd/$repository/archive/master.tar.gz \
 | tar --extract --gzip --strip-components=1
WORKDIR build
RUN bash instructions

# LLLC-server, a go app that manages compilations
ENV repository lllc-server
RUN mkdir --parents $GOPATH/src/github.com/eris-ltd/$repository
COPY . $GOPATH/src/github.com/eris-ltd/$repository
WORKDIR $GOPATH/src/github.com/eris-ltd/$repository/cmd/$repository
RUN go get -d && go install

# Add Eris User
RUN groupadd --system eris && useradd --system --create-home --gid eris eris

## Point to the compiler location.
RUN mkdir --parents /home/eris/.decerver/languages
COPY tests/config.json /home/eris/.decerver/languages/config.json
RUN chown --recursive eris /home/eris/.decerver

USER eris
WORKDIR /home/eris/

EXPOSE 9099
CMD ["lllc-server"]

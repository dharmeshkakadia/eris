FROM golang:1.4
MAINTAINER Eris Industries <support@erisindustries.com>

RUN apt-get update && \
  apt-get install -y \
  libgmp3-dev \
  jq

# Copy In the Good Stuff -- the branch checkout is temporary
RUN git clone https://github.com/eris-ltd/eris-std-lib \
  $GOPATH/src/github.com/eris-ltd/eris-std-lib && \
  cd $GOPATH/src/github.com/eris-ltd/eris-std-lib && \
  git checkout newepm

COPY . $GOPATH/src/github.com/eris-ltd/epm-go/
RUN cd $GOPATH/src/github.com/eris-ltd/epm-go/ && \
  make

# Set a user
ENV user eris
RUN groupadd --system $user && \
  useradd --system --create-home --gid $user $user
COPY *.sh /home/$user/
RUN mkdir /home/$user/genesis/

# User gets the perms (gopath chown necessary for tests)
RUN chown --recursive $user /home/$user && \
  chown --recursive $user $GOPATH/src/github.com/eris-ltd
WORKDIR /home/$user
USER $user
RUN epm init

## How Does It Run?
VOLUME /home/$user/.eris/keys
EXPOSE 15254 15255 15256
CMD ["./start.sh"]

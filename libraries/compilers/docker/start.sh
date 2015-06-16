#!/bin/bash

# Flame out if errors
set -e

# First, set up the certificates. For golang
# to serve over SSL, it must have the domain
# cert, intermediate cert(s), and the root
# cert concatenated into a single cert file
# The gandi certs for *.eris.industries have
# been added to the container (the intermediate
# and the root certificate) so that only the
# end use wildcard cert needs to be added as
# an environment variable.
#
# For other domains, you will have to concatenate
# the certs in the proper order and add that as
# a CERT env variable.
if [ ! -z "$CERT" ]
then
  if [ -f /data/cert.cert ]
  then
    rm /data/cert.crt
  fi
  if [ "$ERIS" = "true" ]
  then
    echo -e "$CERT" >> /data/cert.crt
    cat /data/gandi2.crt >> /data/cert.crt
    cat /data/gandi3.crt >> /data/cert.crt
  else
    echo -e "$CERT" >> /data/cert.crt
  fi
fi

# The SSL private key must be added as an
# environment variable to the container.
if [ ! -z "$KEY" ]
then
  if [ -f /data/key.key ]
  then
    rm /data/key.key
  fi
  echo -e "$KEY" >> /data/key.key
fi

# If either a cert or key has not been added
# then no ssl will be used. Otherwise there are
# two options for the container. If the $SSL_ONLY
# environment variable is set then the container
# will only serve over SSL and will not do an
# http->https redirect. Otherwise the container
# will open both ports and do the redirect.
if [ ! -f /data/cert.crt ] || [ ! -f /data/key.key ]
then
  exec lllc-server --no-ssl --unsecure-port ${UNSECURE_PORT:=9099}
else
  if [ -z $SSL_ONLY ]
  then
    exec lllc-server --unsecure-port ${UNSECURE_PORT:=9099} --secure-port ${SECURE_PORT:=9098} --key /data/key.key --cert /data/cert.crt
  else
    exec lllc-server --secure-only --secure-port ${SECURE_PORT:=9098} --key /data/key.key --cert /data/cert.crt
  fi
fi

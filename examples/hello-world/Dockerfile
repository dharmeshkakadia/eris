FROM eris/decerver:latest
MAINTAINER Eris Industries <support@erisindustries.com>

USER root

COPY . /home/$user/.eris/dapps/helloworld/
RUN chown --recursive $user:$user /home/$user

USER $user

CMD /home/$user/.eris/dapps/helloworld/start.sh

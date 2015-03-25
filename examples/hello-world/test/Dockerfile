FROM node:0.10

# Copy steps from -onbuild because we don't want to run as root.

ENV user node
RUN groupadd --system $user && useradd --system --create-home --gid $user $user

COPY . /home/$user/
WORKDIR /home/$user
RUN chown $user --recursive .
USER $user
RUN npm install

CMD [ "npm", "start" ]

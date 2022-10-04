#! /usr/bin/env sh

docker build -t rtchat .
docker rm -f rtchat
docker run ${docker_opts} -it --net=host --name=rtchat rtchat
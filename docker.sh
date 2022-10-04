#! /usr/bin/env sh

docker build -t rtchat .
docker run ${docker_opts} -it --net=host rtchat
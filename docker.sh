#! /usr/bin/env sh

docker build -t rtchat .
docker run -it --net=host rtchat
#! /usr/bin/env sh

docker build -t rtchat .
docker rm -f rtchat
docker run ${docker_opts} -it \
    --volume=$(pwd)/i2pkeys:/i2pkeys \
    --volume=$(pwd)/onionkeys:/onionkeys \
    --volume=$(pwd)/tlskeys:/tlskeys \
    --net=host --name=rtchat rtchat
docker logs -f rtchat
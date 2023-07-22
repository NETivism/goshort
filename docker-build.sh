#!/bin/bash
echo "Building go binary ..."
go build -o docker/goshort
echo "Done!"

#echo "Building docker image  ..."
#docker build . -t rg.netivism.com.tw/netivism/goshort
#echo "Done!"

#echo "Pushing docker image  ..."
#docker push rg.netivism.com.tw/netivism/goshort
#echo "Done!"


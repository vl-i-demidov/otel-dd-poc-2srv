#!/bin/sh

i=0
while [ $i -lt 300 ]
do
  echo "Request number $i"
  make ping-a
#  sleep 0.1
  i=$(( $i + 1 ))
done
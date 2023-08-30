#!/bin/bash

docker-compose up -d
# start by creating topics on each nsqd node
for i in 'nsq-a' 'nsq-b' 'nsq-c'
do
    docker-compose run -d $i /bin/nsqload -c 1 -r 1 $i:4150 sub-1
    docker-compose run -d $i /bin/nsqload -c 1 -r 1 $i:4150 sub-2
    docker-compose run -d $i /bin/nsqload -c 1 -r 1 $i:4150 sub-3
done
# add fleet of subscibers to each topic
for i in 'nsq-a' 'nsq-b' 'nsq-c'
do
    docker-compose run -d $i /bin/nsqsub -c 200 http://nsqlookupd:4161 sub-1 wg1
    docker-compose run -d $i /bin/nsqsub -c 200 http://nsqlookupd:4161 sub-2 wg1
    docker-compose run -d $i /bin/nsqsub -c 200 http://nsqlookupd:4161 sub-3 wg1
done
# Add the equivalent of 20k messages per second
# 20k / 3 subscriptions / 3 nsqd hosts / 5 publishers each ~= 445 requests per second
for i in 'nsq-a' 'nsq-b' 'nsq-c'
do
    docker-compose run -d $i /bin/nsqload -c 5 -r 445 $i:4150 sub-1
    docker-compose run -d $i /bin/nsqload -c 5 -r 445 $i:4150 sub-2
    docker-compose run -d $i /bin/nsqload -c 5 -r 445 $i:4150 sub-3
done

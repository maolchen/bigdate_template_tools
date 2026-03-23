#!/bin/bash
cd /data/localization/kafka/
nohup bin/zookeeper-server-start.sh config/zookeeper.properties > zk.log &

#!/bin/bash
cd /data/localization/kafka/;
nohup bin/kafka-server-start.sh config/server.properties > kafka.log &

#!/bin/bash
#stop zk
ps aux|grep -v grep|grep config/zookeeper.properties |awk '{print $2}'|xargs kill -9

#!/bin/bash
#stop kafka
ps aux|grep -v grep|grep config/server.properties |awk '{print $2}'|xargs kill -9

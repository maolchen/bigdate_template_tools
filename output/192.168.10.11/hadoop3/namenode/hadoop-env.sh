#!/bin/bash
# Hadoop环境变量配置

# Java Home
export JAVA_HOME=/data/jdk/

# Hadoop配置目录
export HADOOP_CONF_DIR=${HADOOP_HOME}/etc/hadoop

# Hadoop各模块Home
export HADOOP_COMMON_HOME=${HADOOP_HOME}
export HADOOP_HDFS_HOME=${HADOOP_HOME}
export HADOOP_MAPRED_HOME=${HADOOP_HOME}
export HADOOP_YARN_HOME=${HADOOP_HOME}

# JVM参数
export HADOOP_HEAPSIZE_MAX=4096m

# 日志目录
export HADOOP_LOG_DIR=/data/hadoop/logs
export HADOOP_PID_DIR=/data/hadoop/pids

# 用户配置
export HADOOP_IDENT_STRING=bigdata

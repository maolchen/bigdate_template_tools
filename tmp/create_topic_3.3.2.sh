#!/bin/bash
## 4.create and init topic
if [ "$(id -u)" -eq 0 ]; the
  if [ -f /etc/profile ]; then
    source /etc/profile
  fi
els
USER_PROFILE="$HOME/.bash_profile"
  if [ -f "$USER_PROFILE" ]; then
    source "$USER_PROFILE"
  else
    if [ -f "$HOME/.profile" ]; then
      source "$HOME/.profile"
    fi
  fi
fi
cd /data/localization/kafka/

#1.etl-gate
bin/kafka-topics.sh --bootstrap-server realtime-kafka01:9092 --create --topic sdklua_online --partitions=5 --replication-factor 2;
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest gate-consumer.properties sdklua_online;

#2.etl-debug
bin/kafka-topics.sh --bootstrap-server realtime-kafka01:9092 --create --topic debug --partitions=5 --replication-factor 2;
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest debug-consumer.properties debug;

#3.etl-id
bin/kafka-topics.sh --bootstrap-server realtime-kafka01:9092 --create --topic pay_statisv2 --partitions=5 --replication-factor 2;
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest id-consumer.properties pay_statisv2;

#4.etl-count&etl-count-et
bin/kafka-topics.sh --bootstrap-server realtime-kafka01:9092 --create --topic pay_zg_total --partitions=5 --replication-factor 2;
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest count-consumer.properties pay_zg_total;
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest count-et-consumer.properties pay_zg_total;

#5 etl-adv-count(sem)
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK latest consumer-count-adv.properties pay_zg_total

#6.etl-dw
bin/kafka-topics.sh --bootstrap-server realtime-kafka01:9092 --create --topic pay_zg_total_random --partitions=5 --replication-factor 2;
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest consumer-dw.properties pay_zg_total_random;

#7.marketing
#bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK latest consumer-market.properties pay_zg_total_random

#8.es es_index
#bin/kafka-topics.sh --bootstrap-server realtime-kafka01:9092 --create --topic es_index --partitions=5 --replication-factor 2

# v4.3.0新增 标签评价管理
bin/kafka-topics.sh --create   --bootstrap-server realtime-kafka01:9092  --topic tag-evaluate-new   --partitions 1   --replication-factor 1
bin/kafka-topics.sh --create   --bootstrap-server realtime-kafka01:9092  --topic file-task   --partitions 1   --replication-factor 1

# v4.3.6 群组用户导入
bin/kafka-topics.sh --create   --bootstrap-server realtime-kafka01:9092  --topic group-user-load   --partitions 1   --replication-factor 1

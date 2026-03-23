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

#1.mkt-threshold-cep
#bin/kafka-topics.sh --zookeeper localhost:2182 --create --topic mkt-threshold-cep --partitions=5 --replication-factor 2;
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest mkt-threshold-cep.properties mkt-threshold-cep;

#2.mkt-activity-cep
#bin/kafka-topics.sh --zookeeper localhost:2182 --create --topic mkt-activity-cep --partitions=5 --replication-factor 2;
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest mkt-activity-cep.properties mkt-activity-cep;

#3.pay_zg_total_random
#bin/kafka-topics.sh --zookeeper localhost:2182 --create --topic pay_zg_total_random --partitions=5 --replication-factor 2;
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest pay_zg_total_random.properties pay_zg_total_random;

#4.tag_sink_r2p3
#bin/kafka-topics.sh --zookeeper localhost:2182 --create --topic tag_sink_r2p3 --partitions=5 --replication-factor 2;
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest tag_sink_r2p3.properties tag_sink_r2p3;
#bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest count-et-consumer.properties pay_zg_total;

#5 dimension_expansion
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK latest dimension_expansion.properties dimension_expansion

#6.mkt-postman-cep
#bin/kafka-topics.sh --zookeeper localhost:2182 --create --topic mkt-postman-cep --partitions=5 --replication-factor 2;
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest mkt-postman-cep.properties mkt-postman-cep;

#7.mkt-user-cep
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK latest mkt-user-cep.properties mkt-user-cep

#8.mkt-statistics-cep
#bin/kafka-topics.sh --zookeeper localhost:2182 --create --topic mkt-statistics-cep --partitions=5 --replication-factor 2
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest mkt-statistics-cep.properties mkt-statistics-cep;

#9.mkt-batch-cep
#bin/kafka-topics.sh --zookeeper localhost:2182 --create --topic mkt-statistics-cep --partitions=5 --replication-factor 2
bin/kafka-run-class.sh kafka.tools.UpdateOffsetsInZK earliest mkt-batch-cep.properties mkt-batch-cep;

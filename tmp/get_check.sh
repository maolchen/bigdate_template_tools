source  ~/.bash_profile
date "+%Y-%m-%d %H:%M:%S"
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --bootstrap-server realtime-kafka01:9092 --topic sdklua_online --group etl-gate;
date "+%Y-%m-%d %H:%M:%S"
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --bootstrap-server realtime-kafka01:9092 --topic debug --group etl-debug
date "+%Y-%m-%d %H:%M:%S"
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --bootstrap-server realtime-kafka01:9092 --topic pay_statisv2 --group etl-id
date "+%Y-%m-%d %H:%M:%S"
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --bootstrap-server realtime-kafka01:9092 --topic pay_zg_total --group etl-count;
date "+%Y-%m-%d %H:%M:%S"
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --bootstrap-server realtime-kafka01:9092 --topic pay_zg_total --group etl-count-et
date "+%Y-%m-%d %H:%M:%S"
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --bootstrap-server realtime-kafka01:9092 --topic pay_zg_total --group etl-count-adv;
date "+%Y-%m-%d %H:%M:%S"
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --bootstrap-server realtime-kafka01:9092 --topic pay_zg_total_random --group etl-dw;
date "+%Y-%m-%d %H:%M:%S"
find /data/topic_log -name 'tmp_topic_*.log' -mtime +7  | xargs rm -f
echo ""

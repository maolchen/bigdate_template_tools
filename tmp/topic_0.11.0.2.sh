#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic sdklua_online --group etl-gate;
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic debug --group etl-debug
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic pay_statisv2 --group etl-id
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic pay_zg_total --group etl-count;
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic pay_zg_total --group etl-count-et
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic pay_zg_total --group etl-count-adv
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic pay_zg_total_random --group etl-dw;

bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic sdklua_online --group etl-gate 2>/dev/null;
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic debug --group etl-debug 2>/dev/null;
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic pay_statisv2 --group etl-id 2>/dev/null;
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic pay_zg_total_random --group etl-dw 2>/dev/null;
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic pay_zg_total --group etl-count 2>/dev/null;
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic pay_zg_total --group etl-count-et 2>/dev/null;
bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic pay_zg_total --group etl-count-adv 2>/dev/null;

#### CEP相关，按需启用 ####
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic mkt-threshold-cep --group mkt-threshold-cep 2>/dev/null;
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic mkt-activity-cep --group mkt-activity-cep 2>/dev/null;
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic mkt-batch-cep --group mkt-batch-cep 2>/dev/null;
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic pay_zg_total_random --group mkt-adapter-cep 2>/dev/null;
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic tag_sink_r2p3 --group mkt-adapter-cep 2>/dev/null;
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic dimension_expansion --group mkt-adapter-cep 2>/dev/null;
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic mkt-postman-cep --group playwell-mkt-postman-cep 2>/dev/null;
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic mkt-user-cep --group mkt-user-cep 2>/dev/null;
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic mkt-statistics-cep --group mkt-statistics-cep 2>/dev/null;
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic mkt-batch-cep --group mkt-batch-cep 2>/dev/null;

#### 微信服务相关，按需启用 ####
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic wt-weixin-topic --group wt-weixin-group 2>/dev/null
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic zg_see --group etl-zgsee  5>/dev/null
#bin/kafka-run-class.sh kafka.tools.ConsumerOffsetChecker --zookeeper localhost:2182 --topic zg_see_pix --group etl-zgsee-pix 2>/dev/null

<?xml version="1.0"?>
<!--
  Licensed to the Apache Software Foundation (ASF) under one or more
  contributor license agreements.  See the NOTICE file distributed with
  this work for additional information regarding copyright ownership.
  The ASF licenses this file to You under the Apache License, Version 2.0
  (the "License"); you may not use this file except in compliance with
  the License.  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
-->
<?xml-stylesheet type="text/xsl" href="configuration.xsl"?>

<configuration>

<property>
  <name>fs.defaultFS</name>
  <value>hdfs://zhugeio</value>
</property>

<property>  
        <name>hadoop.tmp.dir</name>  
        <value>/data/dfs/tmp</value>  
        <description>A base for other temporary directories.</description>  
</property>  

<property>
  <name>ha.zookeeper.quorum</name>
  <value>{% for host in groups['etl_kafka'] %}{{ host }}:2182{% if not loop.last %},{% endif %}{% endfor %}</value>
</property>

<property>
  <name>io.compression.codecs</name>
  <value>org.apache.hadoop.io.compress.DefaultCodec,org.apache.hadoop.io.compress.GzipCodec,org.apache.hadoop.io.compress.BZip2Codec,com.hadoop.compression.lzo.LzoCodec,com.hadoop.compression.lzo.LzopCodec,org.apache.hadoop.io.compress.SnappyCodec</value>
</property>

<property>
  <name>io.compression.codec.lzo.class</name>
  <value>com.hadoop.compression.lzo.LzoCodec</value>
</property>

<property>
  <name>hadoop.proxyuser.hduser.host</name>
  <value>*</value>
</property>

<property>
  <name>hadoop.proxyuser.hduser.groups</name>
  <value>*</value>
</property>

</configuration>

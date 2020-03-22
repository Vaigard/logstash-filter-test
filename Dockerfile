FROM docker.elastic.co/logstash/logstash:7.5.1

RUN rm /usr/share/logstash/pipeline/logstash.conf

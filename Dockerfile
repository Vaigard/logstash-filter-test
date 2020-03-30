FROM golang:1.7.3
WORKDIR /root
COPY ./http-server/main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

FROM docker.elastic.co/logstash/logstash:7.5.1

EXPOSE 8081

COPY --from=0 /root/server /usr/share/logstash/server

RUN rm /usr/share/logstash/pipeline/logstash.conf

USER root
RUN \
  rpm --rebuilddb && yum clean all && \
  yum install -y epel-release && \
  yum update -y && \
  yum install -y \
                  iproute \
                  python-setuptools \
                  hostname \
                  inotify-tools \
                  yum-utils \
                  which \
                  jq \
                  rsync && \
  yum clean all && \
  easy_install supervisor

COPY ./container/supervisord.conf /etc/supervisor/supervisord.conf
COPY ./container/config/logstash.yml /usr/share/logstash/config/logstash.yml
COPY ./container/config/pipelines.yml /usr/share/logstash/config/pipelines.yml
COPY ./container/pipeline/filter.conf /usr/share/logstash/pipeline/filter.conf
COPY ./container/pipeline/io.conf /usr/share/logstash/pipeline/io.conf

RUN chown logstash:root /usr/share/logstash/server
RUN chown logstash:root /usr/share/logstash/pipeline/filter.conf

ENTRYPOINT ["/usr/bin/supervisord"]

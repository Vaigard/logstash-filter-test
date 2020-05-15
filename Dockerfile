FROM golang:1.13.8
WORKDIR /root
COPY ./http-server/server.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

FROM docker.elastic.co/logstash/logstash:7.5.1

EXPOSE 8181

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

RUN rm /usr/share/logstash/pipeline/logstash.conf

COPY ./README.md /usr/share/logstash/README.md
COPY ./container/supervisord.conf /etc/supervisor/supervisord.conf
COPY ./container/config/* /usr/share/logstash/config/
COPY ./container/pipeline/* /usr/share/logstash/pipeline/

COPY --from=0 /root/server /usr/share/logstash/server

RUN \
  mkdir /usr/share/logstash/patterns && \
  chown logstash:root /usr/share/logstash/server && \
  chown -R logstash:root /usr/share/logstash/pipeline && \
  chown logstash:root /usr/share/logstash/patterns

ENTRYPOINT ["/usr/bin/supervisord"]

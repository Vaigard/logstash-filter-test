FROM golang:1.7.3
WORKDIR /root
COPY ./http-server/main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

FROM docker.elastic.co/logstash/logstash:7.5.1
WORKDIR /usr/share/logstash
COPY --from=0 /root/server .

RUN rm /usr/share/logstash/pipeline/logstash.conf

COPY ./container/config/logstash.yml /usr/share/logstash/config/logstash.yml
COPY ./container/config/pipelines.yml /usr/share/logstash/config/pipelines.yml

COPY ./container/io/filter.conf /usr/share/logstash/pipeline/filter.conf
COPY ./container/io/io.conf /usr/share/logstash/pipeline/io.conf

CMD ["./server"]

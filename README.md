# logstash-filter-test
Http-server testing Logstash filters

Works on port 8081.

Pages:
- /ping
- /upload

Example of using:
```
./http-server/client.sh -s 127.0.0.1:8081 -f ./filter.txt -m ./message.txt
```

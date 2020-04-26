# logstash-filter-test
HTTP-server testing Logstash filters

Works on port 8181.

Example of start container:
```
docker run -it -p 8181:8181 -v /home/user/server.log:/usr/share/logstash/server.log --name logstash-test-server logstash-test-image
```

Example of using within client script:
```
./client.sh -s 127.0.0.1:8181 -f ./filter.txt -m ./message.txt
```

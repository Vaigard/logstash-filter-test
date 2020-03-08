# logstash-filter-test
Http-server testing Logstash filters

Works on port 8081.

Pages:
- /ping
- /upload

Example of upload request:
```
curl -i --request POST -F "filter=@/home/user/filter.conf" -F 'message="testhost testtext"' -F 'expected="{\"name\":\"hostname\",\"message\":\"testtext\"}"' 127.0.0.1:8081/upload
```

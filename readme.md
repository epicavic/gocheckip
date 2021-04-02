## simple http ip checker

```bash
$ UPDATE_INTERVAL=10s UPDATE_IPV4_URL='https://www.cloudflare.com/ips-v4' go run .
2021/04/02 15:44:17 Update interval is set to: 10s
2021/04/02 15:44:17 Update IPv4 URL is set to: https://www.cloudflare.com/ips-v4
2021/04/02 15:44:17 Starting server on: localhost:8080

$ curl -i -w'\n' "localhost:8080/"
HTTP/1.1 200 OK
Date: Fri, 02 Apr 2021 12:50:25 GMT
Content-Length: 246
Content-Type: text/plain; charset=utf-8

["162.158.0.0/15","172.64.0.0/13","173.245.48.0/20","103.21.244.0/22","103.22.200.0/22","103.31.4.0/22","188.114.96.0/20","198.41.128.0/17","197.234.240.0/22","104.16.0.0/12","141.101.64.0/18","190.93.240.0/20","131.0.72.0/22","108.162.192.0/18"]
```

```bash
$ curl -i -w'\n' "localhost:8080/check"
HTTP/1.1 400 Bad Request
Date: Fri, 02 Apr 2021 12:45:38 GMT
Content-Length: 45
Content-Type: text/plain; charset=utf-8

X-Real-IP header is not provided or malformed
```

```bash
$ curl -i -w'\n' -H 'X-Real-IP: 1.2.3.4' "localhost:8080/check"
HTTP/1.1 200 OK
Date: Fri, 02 Apr 2021 12:44:31 GMT
Content-Length: 21
Content-Type: text/plain; charset=utf-8

{"real_ip":"1.2.3.4"}
```

```bash
$ curl -i -w'\n' -H 'X-Real-IP: 103.21.244.1' "localhost:8080/check"
HTTP/1.1 503 Service Unavailable
Date: Fri, 02 Apr 2021 12:45:32 GMT
Content-Length: 26
Content-Type: text/plain; charset=utf-8

{"real_ip":"103.21.244.1"}
```

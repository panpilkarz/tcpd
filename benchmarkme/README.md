Test throughput and latency using ApacheBench

```
make

./benchmarkme # runs super simple http server using gotcpd

yum install httpd-tools # install ApacheBench

./ab -n 500000 -c 200 -k http://localhost:8000/ 
```

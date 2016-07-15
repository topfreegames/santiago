Santiago's Benchmarks
=====================

You can see santiago's benchmarks in our [CI server](https://travis-ci.org/topfreegames/santiago/) as they get run with every build.

## Results

Runnning with Apache Benchmark on a Macbook Pro, with this command:

    $ ab -n 10000 -c 30 -p ab.data "http://127.0.0.1:3333/hooks?method=POST&url=http%3A//127.0.0.1:3000/hooks/"

With the ab.data file containing:

    hello=world

The results should be similar to these:

```
This is ApacheBench, Version 2.3 <$Revision: 1706008 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking 127.0.0.1 (be patient)
Completed 1000 requests
Completed 2000 requests
Completed 3000 requests
Completed 4000 requests
Completed 5000 requests
Completed 6000 requests
Completed 7000 requests
Completed 8000 requests
Completed 9000 requests
Completed 10000 requests
Finished 10000 requests


Server Software:        iris
Server Hostname:        127.0.0.1
Server Port:            3333

Document Path:          /hooks?method=POST&url=http%3A//10.0.23.64:3000/hooks/
Document Length:        2 bytes

Concurrency Level:      30
Time taken for tests:   2.034 seconds
Complete requests:      10000
Failed requests:        0
Total transferred:      1510000 bytes
Total body sent:        1940000
HTML transferred:       20000 bytes
Requests per second:    4916.58 [#/sec] (mean)
Time per request:       6.102 [ms] (mean)
Time per request:       0.203 [ms] (mean, across all concurrent requests)
Transfer rate:          725.00 [Kbytes/sec] received
                        931.46 kb/s sent
                        1656.47 kb/s total

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    2   0.7      2       9
Processing:     1    4   1.3      4      16
Waiting:        0    4   1.3      4      16
Total:          1    6   1.6      6      18

Percentage of the requests served within a certain time (ms)
  50%      6
  66%      6
  75%      6
  80%      6
  90%      8
  95%      9
  98%     13
  99%     13
 100%     18 (longest request)
```

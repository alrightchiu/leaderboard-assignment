## 需求

實作 Leaderboard 功能，開出兩支 API：
- POST：更新參賽者的分數
- GET：取得前十名參賽者資訊


## 啟動

使用 docker compose 可以在本地端建立服務

```
$ docker compose up
```

加入 nginx 之前的版本 port 開在 80，之後在 8080（どうして）


```
# POST
curl http://0.0.0.0:8080/api/v1/score -X POST -H "Content-Type:application/json" -H "ClientId:user-1" -d '{"score":666}'

# GET
curl http://0.0.0.0:8080/api/v1/leaderboard
```

## 說明

一共實作了四種策略，並且以 [rakyll/hey](https://github.com/rakyll/hey) 做效能測試
1. 使用 PostgreSQL(搭配 gorm)實作（8fbf343）
2. 使用 Redis(搭配 go-redis)實作（4fa543b）
3. 延續 2，加入 nginx 並模擬兩台 rest server（1177640）
4. 延續 3，加入一個 redis replica，只針對 master 寫入，master 及 replica 皆可讀取（5912fe7）

### hey 簡易說明

> hey is a tiny program that sends some load to a web application.

指令：`hey -n 10000 -c 500 -t 10 http://0.0.0.0/api/v1/leaderboard`
- number of request：10000
- concurrency level：500
- timeout：10s


### 策略 1

![strategy_1](/img/s_1.png)

commit: 8fbf343

指令：`hey -n 10000 -c 500 -t 10 http://0.0.0.0/api/v1/leaderboard`
- DB 參數 MaxIdleConns 使用預設值 2，GET 同時沒有 POST
  - 結果：28% 回 200，69.9% 超過 10s timeout，[report](/hey-report/postgresql_c_500_n_10000_t_10.txt)
- DB 參數 MaxIdleConns 使用預設值 10，GET 同時沒有 POST
  - 結果：41% 回 200，55% 超過 10s timeout，[report](/hey-report/postgresql_c_500_n_10000_t_10_idleconn_10.txt)
- DB 參數 MaxIdleConns 使用預設值 2，GET 同時每 200ms 打一次 POST
  - 結果：29% 回 200，69% 超過 10s timeout，[report](/hey-report/postgresql_c_500_n_10000_t_10_rw.txt)

指令：`hey -n 100000 -c 1000 -t 10 http://0.0.0.0/api/v1/leaderboard`
- DB 參數 MaxIdleConns 使用預設值 2，GET 同時沒有 POST
  - 結果：16.1% 回 200，26% 回 500，50% 超過 10s timeout，[report](/hey-report/postgresql_c_1000_n_100000_t_10.txt)


### 策略 2

![strategy_2](/img/s_2.png)

commit: 4fa543b

將 PostgreSQL 替換成 Redis，，試圖提高 API 成功率

指令：`hey -n 100000 -c 1000 -t 10 http://0.0.0.0/api/v1/leaderboard`
- GET 同時沒有 POST
  - 結果：100% 回 200，平均 0.55s，RPS 1.7k，[report](/hey-report/redis_c_1000_n_100000_t_10.txt)
- GET 同時每 200ms 打一次 POST
  - 結果：100% 回 200，平均 0.54s，RPS 1.8k，[report](/hey-report/redis_c_1000_n_100000_t_10_rw_200ms.txt)
- GET 同時每 10ms 打一次 POST
  - 結果：100% 回 200，平均 0.83s，RPS 1.1k，[report](/hey-report/redis_c_1000_n_100000_t_10_rw_10ms.txt)



### 策略 3


![strategy_3](/img/s_3.png)

commit: 1177640

延續策略 2，加入 nginx 模擬兩台 rest server 的情境，結果比只有一台 rest server 差

指令：`hey -n 100000 -c 1000 -t 10 http://0.0.0.0:8080/api/v1/leaderboard`
- GET 同時沒有 POST
  - 結果：90% 回 200，9% timeout，平均 1.29 s，RPS 383，[report](/hey-report/redis_c_1000_n_100000_t_10_nginx_server_2.txt)



### 策略 4


![strategy_4](/img/s_4.png)

commit: 5912fe7

延續策略 3，多加一個 replica redis，寫入只針對 master redis，讀取以 3/7 對 master 4/7 對 replica 做分配（magic number）

指令：`hey -n 100000 -c 1000 -t 10 http://0.0.0.0:8080/api/v1/leaderboard`
- GET 同時沒有 POST
  - 結果#1：90% 回 200，9.5% timeout，平均 1.34 s，RPS 372，[report](/hey-report/redis_c_1000_n_100000_t_10_nginx_server_2_replica_1.txt)
  - 結果#2：91% 回 200，8.2% timeout，平均 1.39 s，RPS 341，[report](/hey-report/redis_c_1000_n_100000_t_10_nginx_server_2_replica_1_test_2.txt)


## 結論

Redis 實作在 memory，會比在 disk 的 PostgreSQL 快很多，適合同時間大流量的場景。

但是當有多台 server 要讀取同一個 redis 時，似乎也會讓效能下降（有待研究）。

另外，嘗試開 redis replica 分散讀取 request 的策略效果不佳，有可能是用法有誤（透過 redis-cli 的 `monitor` 觀察到，在寫入 master 時，在極短時間內就會把資料同步到 replica 上），沒有試到 cluster 的做法，也許是一種解決方法（？）



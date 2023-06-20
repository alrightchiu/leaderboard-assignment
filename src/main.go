package main

import (
	"context"
	"fmt"
	"leaderboard/constant"
	"leaderboard/logger"
	"leaderboard/redis"
	"leaderboard/rest"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func main() {
	logger := logger.Get()
	ctx := context.Background()

	redisMasterClient := redis.NewMasterClient()
	defer redisMasterClient.Close()
	redisReplicaClient := redis.NewReplicaClient()
	defer redisReplicaClient.Close()

	cronJob := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	cronJobDuration := fmt.Sprintf("@every %dm", constant.CronResetLeaderBoardDurationInMin)
	cronJob.AddFunc(cronJobDuration, func() {
		if err := redisMasterClient.Del(ctx, redis.KeyPlayers); err != nil {
			logger.Errorf("err: %+v", err)
		} else {
			logger.Infof("reset redis key(%s) done", redis.KeyPlayers)
		}
	})
	cronJob.Start()
	defer cronJob.Stop()

	engine := gin.Default()
	rest.RegisterHandler(engine, redisMasterClient, redisReplicaClient)
	rest.ListenAndServe(engine)
}

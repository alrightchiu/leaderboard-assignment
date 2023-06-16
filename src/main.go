package main

import (
	"context"
	"fmt"
	"leader-board/constant"
	"leader-board/logger"
	"leader-board/redis"
	"leader-board/rest"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func main() {
	logger := logger.Get()
	ctx := context.Background()

	redisClient := redis.NewClient(nil)
	defer redisClient.Close()

	cronJob := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	cronJobDuration := fmt.Sprintf("@every %dm", constant.CronResetLeaderBoardDurationInMin)
	cronJob.AddFunc(cronJobDuration, func() {
		if err := redisClient.Del(ctx, redis.KeyPlayers); err != nil {
			logger.Errorf("err: %+v", err)
		} else {
			logger.Infof("reset redis key(%s) done", redis.KeyPlayers)
		}
	})
	cronJob.Start()
	defer cronJob.Stop()

	engine := gin.Default()
	rest.RegisterHandler(engine, redisClient)
	rest.ListenAndServe(engine)
}

package main

import (
	"context"
	"fmt"
	"leader-board/constant"
	"leader-board/dao"
	"leader-board/db"
	"leader-board/logger"
	"leader-board/rest"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func main() {
	logger := logger.Get()
	ctx := context.Background()
	db, err := db.NewDB()
	if err != nil {
		logger.Panic(err)
	}
	logger.Info("db connected")

	playerDao, err := dao.NewPlayerDao(db)
	if err != nil {
		logger.Panic(err)
	}

	cronJob := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	cronJobDuration := fmt.Sprintf("@every %dm", constant.CronResetLeaderBoardDurationInMin)
	cronJob.AddFunc(cronJobDuration, func() {
		if err := playerDao.Truncate(ctx); err != nil {
			logger.Errorf("err: %+v\n", err)
		} else {
			logger.Info("reset table players done")
		}
	})
	cronJob.Start()
	defer cronJob.Stop()

	engine := gin.Default()
	rest.RegisterHandler(engine, playerDao)
	rest.ListenAndServe(engine)
}

package rest

import (
	"context"
	"fmt"
	"leaderboard/logger"
	"leaderboard/redis"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	restPort = os.Getenv("REST_PORT")

	getRedisClientFn = getRedisClient
)

type TopPlayer struct {
	ClientID string  `json:"clientId"`
	Score    float64 `json:"score"`
}

type LeaderResp struct {
	TopPlayers []*TopPlayer `json:"topPlayers"`
}

type GetQuery struct {
	Limit int `json:"limit" form:"limit,default=10"`
}

type PostHeader struct {
	ClientID string `header:"clientId" binding:"required"`
}

type PostBody struct {
	Score float64 `json:"score" binding:"required"`
}

type impl struct {
	logger             logger.Logger
	redisMasterClient  redis.Redis
	redisReplicaClient redis.Redis
}

func RegisterHandler(
	ginEngine *gin.Engine,
	redisMasterClient redis.Redis,
	redisReplicaClient redis.Redis,
) {
	rest := &impl{
		logger:             logger.Get(),
		redisMasterClient:  redisMasterClient,
		redisReplicaClient: redisReplicaClient,
	}

	v1 := ginEngine.Group("/api/v1")
	{
		v1.POST("/score", rest.assignScore)
		v1.GET("/leaderboard", rest.getLeaders)
	}
}

func ListenAndServe(
	ginEngine *gin.Engine,
) {
	logger := logger.Get()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", restPort),
		Handler: ginEngine,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("listen: %s\n", err)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify player of shutdown.
	stop()
	logger.Info("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown: ", err)
	}

	logger.Info("Server exiting")
}

func (i *impl) assignScore(c *gin.Context) {
	var header PostHeader
	if err := c.ShouldBindHeader(&header); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var body PostBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := i.redisMasterClient.ZAdd(c, redis.KeyPlayers, header.ClientID, body.Score); err != nil {
		i.logger.Errorf("redisMasterClient.ZAdd err: %+v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (i *impl) getLeaders(c *gin.Context) {
	var params GetQuery
	if err := c.ShouldBindQuery(&params); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	limit := params.Limit

	redisClient := getRedisClientFn(i.redisMasterClient, i.redisReplicaClient)

	leaders, err := redisClient.ZRevRangeWithScores(c, redis.KeyPlayers, 0, int64(limit-1))
	if err != nil {
		i.logger.Errorf("redisClient.ZRevRangeWithScores err: %+v", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	topPlayers := make([]*TopPlayer, len(leaders))
	for i := 0; i < len(leaders); i++ {
		topPlayers[i] = &TopPlayer{
			ClientID: leaders[i].Member.(string),
			Score:    leaders[i].Score,
		}
	}

	res := LeaderResp{
		TopPlayers: topPlayers,
	}

	c.JSON(http.StatusOK, res)
}

// 3/7 use master, 4/7 use replica
func getRedisClient(redisMasterClient, redisReplicaClient redis.Redis) redis.Redis {
	count := 7
	randNum := rand.Intn(count)
	if randNum%count < 3 {
		return redisMasterClient
	}
	return redisReplicaClient
}

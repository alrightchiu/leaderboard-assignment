package rest

import (
	"context"
	"leader-board/dao"
	"leader-board/logger"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
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
	playerDao dao.PlayerDao
	logger    logger.Logger
}

func RegisterHandler(
	ginEngine *gin.Engine,
	PlayerDao dao.PlayerDao,
) {
	rest := &impl{
		playerDao: PlayerDao,
		logger:    logger.Get(),
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
		Addr:    ":80",
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

	player := &dao.Player{
		ClientID: header.ClientID,
		Score:    body.Score,
	}

	_, err := i.playerDao.Upsert(c, player)
	if err != nil {
		i.logger.Errorf("playerDao.Upsert err: %+v", err)
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

	leaders, err := i.playerDao.GetTopN(c, limit)
	if err != nil {
		i.logger.Errorf("playerDao.GetTopN err: %+v", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	topPlayers := make([]*TopPlayer, len(leaders))
	for i := 0; i < len(leaders); i++ {
		topPlayers[i] = &TopPlayer{
			ClientID: leaders[i].ClientID,
			Score:    leaders[i].Score,
		}
	}

	res := LeaderResp{
		TopPlayers: topPlayers,
	}

	c.JSON(http.StatusOK, res)
}

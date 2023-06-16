package redis

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
)

const (
	testKeyPlayers = "leaderboard:testPlayers"
)

func TestPlayerDaoTestSuite(t *testing.T) {
	suite.Run(t, new(redisTestSuite))
}

type redisTestSuite struct {
	suite.Suite
	client Redis
	ctx    context.Context
}

func (s *redisTestSuite) SetupSuite() {
	s.client = NewClient(nil)
	s.ctx = context.TODO()
}

func (s *redisTestSuite) SetupTest() {
}

func (s *redisTestSuite) TearDownSuite() {
}

func (s *redisTestSuite) Test() {
	members := []redis.Z{
		{Member: "t-mac", Score: 90},
		{Member: "kobe", Score: 88},
		{Member: "kd", Score: 85},
		{Member: "dw", Score: 90},
		{Member: "shaq", Score: 84},
		{Member: "kp", Score: 0},
	}
	for _, m := range members {
		err := s.client.ZAdd(s.ctx, testKeyPlayers, m.Member.(string), m.Score)
		s.NoError(err)
	}

	result, err := s.client.ZRevRangeWithScores(s.ctx, testKeyPlayers, 0, 4)
	s.NoError(err)
	s.Equal(5, len(result))
	s.Equal(members[0].Member, result[0].Member)
	s.Equal(members[1].Member, result[2].Member)

	// update
	members[2].Score = 89
	err = s.client.ZAdd(s.ctx, testKeyPlayers, members[2].Member.(string), members[2].Score)
	s.NoError(err)
	result, err = s.client.ZRevRangeWithScores(s.ctx, testKeyPlayers, 0, 4)
	s.NoError(err)
	s.Equal(5, len(result))
	s.Equal(members[2].Member, result[2].Member)

	// reset
	err = s.client.Del(s.ctx, testKeyPlayers)
	s.NoError(err)
	result, err = s.client.ZRevRangeWithScores(s.ctx, testKeyPlayers, 0, -1)
	s.NoError(err)
	s.Equal(0, len(result))

	// set same key again
	members = []redis.Z{
		{Member: "t-mac", Score: 90},
		{Member: "kobe", Score: 88},
	}
	for _, m := range members {
		err := s.client.ZAdd(s.ctx, testKeyPlayers, m.Member.(string), m.Score)
		s.NoError(err)
	}

	result, err = s.client.ZRevRangeWithScores(s.ctx, testKeyPlayers, 0, -1)
	s.NoError(err)
	s.Equal(2, len(result))
}

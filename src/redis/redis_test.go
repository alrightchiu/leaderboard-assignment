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
	master  Redis
	replica Redis
	ctx     context.Context
}

func (s *redisTestSuite) SetupSuite() {
	s.master = NewMasterClient()
	s.replica = NewReplicaClient()
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
		err := s.master.ZAdd(s.ctx, testKeyPlayers, m.Member.(string), m.Score)
		s.NoError(err)
	}

	// master
	resultMaster, err := s.master.ZRevRangeWithScores(s.ctx, testKeyPlayers, 0, 4)
	s.NoError(err)
	s.Equal(5, len(resultMaster))
	s.Equal(members[0].Member, resultMaster[0].Member)
	s.Equal(members[1].Member, resultMaster[2].Member)

	resultReplica, err := s.replica.ZRevRangeWithScores(s.ctx, testKeyPlayers, 0, 4)
	s.NoError(err)
	s.Equal(5, len(resultReplica))
	s.Equal(members[0].Member, resultReplica[0].Member)
	s.Equal(members[1].Member, resultReplica[2].Member)

	s.Equal(resultMaster, resultReplica)

	// update
	members[2].Score = 89
	err = s.master.ZAdd(s.ctx, testKeyPlayers, members[2].Member.(string), members[2].Score)
	s.NoError(err)
	resultMaster, err = s.master.ZRevRangeWithScores(s.ctx, testKeyPlayers, 0, 4)
	s.NoError(err)
	s.Equal(5, len(resultMaster))
	s.Equal(members[2].Member, resultMaster[2].Member)

	// reset by master, check reset by replica
	err = s.master.Del(s.ctx, testKeyPlayers)
	s.NoError(err)
	resultReplica, err = s.replica.ZRevRangeWithScores(s.ctx, testKeyPlayers, 0, -1)
	s.NoError(err)
	s.Equal(0, len(resultReplica))

	// set same key again
	members = []redis.Z{
		{Member: "t-mac", Score: 90},
		{Member: "kobe", Score: 88},
	}
	for _, m := range members {
		err := s.master.ZAdd(s.ctx, testKeyPlayers, m.Member.(string), m.Score)
		s.NoError(err)
	}

	resultMaster, err = s.master.ZRevRangeWithScores(s.ctx, testKeyPlayers, 0, -1)
	s.NoError(err)
	s.Equal(2, len(resultMaster))
}

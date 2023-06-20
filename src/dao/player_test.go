package dao

import (
	"context"
	"leaderboard/db"
	"testing"

	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestPlayerDaoTestSuite(t *testing.T) {
	suite.Run(t, new(playerDaoTestSuite))
}

type playerDaoTestSuite struct {
	suite.Suite
	db        *gorm.DB
	ctx       context.Context
	PlayerDao PlayerDao
}

func (s *playerDaoTestSuite) SetupSuite() {
	var err error
	s.db, err = db.NewTestDatabase()
	s.Require().NoError(err)

	s.ctx = context.TODO()

	PlayerDao, err := NewPlayerDao(s.db)
	s.Require().NoError(err)
	s.PlayerDao = PlayerDao
}

func (s *playerDaoTestSuite) SetupTest() {
}

func (s *playerDaoTestSuite) TearDownTest() {
	err := s.PlayerDao.Truncate(s.ctx)
	s.NoError(err)
}

func (s *playerDaoTestSuite) TearDownSuite() {
	s.db.Migrator().DropTable(&Player{})
}

func (s *playerDaoTestSuite) TestUpsert() {
	// case: create new
	user1 := &Player{
		ClientID: "user-upsert-1",
		Score:    1,
	}
	result, err := s.PlayerDao.Upsert(s.ctx, user1)
	s.NoError(err)
	s.Equal(user1.ClientID, result.ClientID)
	s.Equal(user1.Score, result.Score)

	user2 := &Player{
		ClientID: "user-upsert-2",
		Score:    2,
	}
	result, err = s.PlayerDao.Upsert(s.ctx, user2)
	s.NoError(err)
	s.Equal(user2.ClientID, result.ClientID)
	s.Equal(user2.Score, result.Score)

	// case: update score
	user1.Score = 6
	result, err = s.PlayerDao.Upsert(s.ctx, user1)
	s.NoError(err)
	s.Equal(user1.ClientID, result.ClientID)
	s.Equal(user1.Score, result.Score)
}

func (s *playerDaoTestSuite) TestGetTopN() {
	users := []*Player{
		{
			ClientID: "user-topn-1",
			Score:    4.77,
		},
		{
			ClientID: "user-topn-2",
			Score:    9.93,
		},
		{
			ClientID: "user-topn-3",
			Score:    7.0,
		},
		{
			ClientID: "user-topn-4",
			Score:    2.6,
		},
	}
	for _, u := range users {
		_, _ = s.PlayerDao.Upsert(s.ctx, u)
	}

	limit := 10
	result, err := s.PlayerDao.GetTopN(s.ctx, limit)
	s.NoError(err)
	s.Equal(len(users), len(result))
	s.Equal(users[1].ClientID, result[0].ClientID)
	s.Equal(users[0].ClientID, result[2].ClientID)
}

func (s *playerDaoTestSuite) TestTruncate() {
	user := &Player{
		ClientID: "user-truncate",
		Score:    1,
	}
	_, _ = s.PlayerDao.Upsert(s.ctx, user)
	limit := 10
	result, _ := s.PlayerDao.GetTopN(s.ctx, limit)
	s.Equal(1, len(result))

	err := s.PlayerDao.Truncate(s.ctx)
	s.NoError(err)

	result, _ = s.PlayerDao.GetTopN(s.ctx, limit)
	s.Equal(0, len(result))
}

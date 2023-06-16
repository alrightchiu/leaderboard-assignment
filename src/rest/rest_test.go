package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"leader-board/redis"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/suite"
)

const (
	testAssignScoreURL = "/api/v1/score"
	testGetLeadersURL  = "/api/v1/leaderboard"

	testKeyPlayers = redis.KeyPlayers
)

func TestRestTestSuite(t *testing.T) {
	suite.Run(t, new(restTestSuite))
}

type restTestSuite struct {
	suite.Suite
	ginEngine *gin.Engine
	mockRedis redismock.ClientMock
}

func (s *restTestSuite) SetupSuite() {
	mockRedisClient, mockRedis := redismock.NewClientMock()
	s.mockRedis = mockRedis
	redisClient := redis.NewClient(mockRedisClient)

	gin.SetMode(gin.TestMode)
	server := gin.Default()
	RegisterHandler(server, redisClient)
	s.ginEngine = server
}

func (s *restTestSuite) SetupTest() {
}

func (s *restTestSuite) TearDownSuite() {
	s.NoError(s.mockRedis.ExpectationsWereMet())
}

func (s *restTestSuite) request(
	method string,
	url string,
	headers map[string]string,
	body interface{},
) (*httptest.ResponseRecorder, error) {
	bs, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(bs))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	w := httptest.NewRecorder()
	s.ginEngine.ServeHTTP(w, req)

	return w, nil
}

func (s *restTestSuite) TestAssignScore() {
	resURL, err := url.Parse(testAssignScoreURL)
	s.NoError(err)

	reqHeader := map[string]string{
		"clientId": "user-1",
	}
	reqBody := map[string]interface{}{
		"score": 77.5,
	}
	player := redis.Z{
		Member: reqHeader["clientId"],
		Score:  reqBody["score"].(float64),
	}
	s.mockRedis.ExpectZAdd(testKeyPlayers, player).SetVal(1)

	res, err := s.request("POST", resURL.String(), reqHeader, reqBody)
	s.NoError(err)
	s.Equal(http.StatusOK, res.Code)

	// error 500
	s.mockRedis.ExpectZAdd(testKeyPlayers, player).SetErr(errors.New("something went wrong"))

	res, err = s.request("POST", resURL.String(), reqHeader, reqBody)
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, res.Code)

	// error 400 invalid header
	reqHeader = map[string]string{
		"wrong-cleint-id": "user-1",
	}

	res, err = s.request("POST", resURL.String(), reqHeader, reqBody)
	s.NoError(err)
	s.Equal(http.StatusBadRequest, res.Code)

	// error 400 invalid body
	reqHeader = map[string]string{
		"clientId": "user-1",
	}
	reqBody = map[string]interface{}{
		"wrong-score": 77.5,
	}

	res, err = s.request("POST", resURL.String(), reqHeader, reqBody)
	s.NoError(err)
	s.Equal(http.StatusBadRequest, res.Code)

}

func (s *restTestSuite) TestGetLeaders() {
	resURL, err := url.Parse(testGetLeadersURL)
	s.NoError(err)

	// response empty array
	limit := 10
	s.mockRedis.ExpectZRevRangeWithScores(testKeyPlayers, 0, int64(limit-1)).SetVal([]redis.Z{})

	res, err := s.request("GET", resURL.String(), nil, nil)
	s.NoError(err)
	s.Equal(http.StatusOK, res.Code)

	// response 2 element with query string
	limit = 5
	query := resURL.Query()
	query.Add("limit", fmt.Sprint(limit))
	query.Add("unknown", "hi")
	resURL.RawQuery = query.Encode()

	mockPlayers := []redis.Z{
		{
			Member: "player-1",
			Score:  1,
		},
		{
			Member: "player-2",
			Score:  2,
		},
	}
	s.mockRedis.ExpectZRevRangeWithScores(testKeyPlayers, 0, int64(limit-1)).SetVal(mockPlayers)

	res, err = s.request("GET", resURL.String(), nil, nil)
	s.NoError(err)
	s.Equal(http.StatusOK, res.Code)
	bs, err := io.ReadAll(res.Body)
	s.NoError(err)
	result := LeaderResp{}
	s.NoError(json.Unmarshal(bs, &result))
	s.Equal(len(mockPlayers), len(result.TopPlayers))

	// error 500
	s.mockRedis.ExpectZRevRangeWithScores(testKeyPlayers, 0, int64(limit-1)).SetErr(errors.New("something went wrong"))

	res, err = s.request("GET", resURL.String(), nil, nil)
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, res.Code)
}

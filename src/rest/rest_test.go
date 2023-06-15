package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"leader-board/dao"
	daomocks "leader-board/dao/mocks"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	testAssignScoreURL = "/api/v1/score"
	testGetLeadersURL  = "/api/v1/leaderboard"
)

func TestRestTestSuite(t *testing.T) {
	suite.Run(t, new(restTestSuite))
}

type restTestSuite struct {
	suite.Suite
	ginEngine     *gin.Engine
	mockPlayerDao *daomocks.PlayerDao
}

func (s *restTestSuite) SetupSuite() {
	s.mockPlayerDao = daomocks.NewPlayerDao(s.T())
	gin.SetMode(gin.TestMode)
	server := gin.Default()
	RegisterHandler(server, s.mockPlayerDao)
	s.ginEngine = server
}

func (s *restTestSuite) SetupTest() {
}

func (s *restTestSuite) TearDownSuite() {
	s.mockPlayerDao.AssertExpectations(s.T())
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
	player := &dao.Player{
		ClientID: reqHeader["clientId"],
		Score:    reqBody["score"].(float64),
	}
	s.mockPlayerDao.EXPECT().Upsert(mock.Anything, player).Return(nil, nil).Once()

	res, err := s.request("POST", resURL.String(), reqHeader, reqBody)
	s.NoError(err)
	s.Equal(http.StatusOK, res.Code)

	// error 500
	s.mockPlayerDao.EXPECT().Upsert(mock.Anything, player).Return(nil, errors.New("something went wrong")).Once()

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
	s.mockPlayerDao.EXPECT().GetTopN(mock.Anything, limit).Return([]*dao.Player{}, nil).Once()

	res, err := s.request("GET", resURL.String(), nil, nil)
	s.NoError(err)
	s.Equal(http.StatusOK, res.Code)

	// response 2 element with query string
	limit = 5
	query := resURL.Query()
	query.Add("limit", fmt.Sprint(limit))
	query.Add("unknown", "hi")
	resURL.RawQuery = query.Encode()

	mockPlayers := []*dao.Player{
		{
			ClientID: "player-1",
			Score:    1,
		},
		{
			ClientID: "player-2",
			Score:    2,
		},
	}
	s.mockPlayerDao.EXPECT().GetTopN(mock.Anything, limit).Return(mockPlayers, nil).Once()

	res, err = s.request("GET", resURL.String(), nil, nil)
	s.NoError(err)
	s.Equal(http.StatusOK, res.Code)
	bs, err := io.ReadAll(res.Body)
	s.NoError(err)
	result := LeaderResp{}
	s.NoError(json.Unmarshal(bs, &result))
	s.Equal(len(mockPlayers), len(result.TopPlayers))

	// error 500
	s.mockPlayerDao.EXPECT().GetTopN(mock.Anything, limit).Return(nil, errors.New("something went wrong")).Once()

	res, err = s.request("GET", resURL.String(), nil, nil)
	s.NoError(err)
	s.Equal(http.StatusInternalServerError, res.Code)
}

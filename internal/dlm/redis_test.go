package dlm_test

import (
	"testing"
	"time"

	"github.com/rivalry-matchmaker/rivalry/internal/dlm"
	goredislib "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/stvp/tempredis"
)

type DLMTestSuite struct {
	suite.Suite
	server *tempredis.Server
	dlm    dlm.DLM
}

func (suite *DLMTestSuite) SetupTest() {
	var err error
	suite.server, err = tempredis.Start(tempredis.Config{})
	require.NoError(suite.T(), err)
	suite.dlm = dlm.NewRedisDLM("prefix", &goredislib.Options{
		Network: "unix",
		Addr:    suite.server.Socket(),
	})
}

func (suite *DLMTestSuite) TestLockUnlock() {
	name := "name"
	for i := 0; i < 10; i++ {
		require.NoError(suite.T(), suite.dlm.Lock(name, time.Second/2))
		_, err := suite.dlm.Unlock(name)
		require.NoError(suite.T(), err)
	}
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(DLMTestSuite))
}

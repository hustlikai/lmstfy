package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bitleak/lmstfy/auth"
	"github.com/bitleak/lmstfy/config"
	redis_engine "github.com/bitleak/lmstfy/engine/redis"
	"github.com/bitleak/lmstfy/helper"
	"github.com/bitleak/lmstfy/push"
	"github.com/bitleak/lmstfy/server/handlers"
	"github.com/bitleak/lmstfy/throttler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func ginTest(req *http.Request) (*gin.Context, *gin.Engine, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	gin.SetMode(gin.ReleaseMode)
	ctx, engine := gin.CreateTestContext(w)
	ctx.Request = req
	return ctx, engine, w
}

var (
	CONF *config.Config
)

func init() {
	cfg := os.Getenv("LMSTFY_TEST_CONFIG")
	if cfg == "" {
		panic(`
############################################################
PLEASE setup env LMSTFY_TEST_CONFIG to the config file first
############################################################
`)
	}
	var err error
	if CONF, err = config.MustLoad(os.Getenv("LMSTFY_TEST_CONFIG")); err != nil {
		panic(fmt.Sprintf("Failed to load config file: %s", err))
	}
}

func setup() {
	logger := logrus.New()
	level, _ := logrus.ParseLevel(CONF.LogLevel)
	logger.SetLevel(level)

	for _, poolConf := range CONF.Pool {
		conn := helper.NewRedisClient(&poolConf, nil)
		err := conn.Ping().Err()
		if err != nil {
			panic(fmt.Sprintf("Failed to ping: %s", err))
		}
		err = conn.FlushDB().Err()
		if err != nil {
			panic(fmt.Sprintf("Failed to flush db: %s", err))
		}
	}

	if err := redis_engine.Setup(CONF, logger); err != nil {
		panic(fmt.Sprintf("Failed to setup redis engine: %s", err))
	}

	if err := auth.Setup(CONF); err != nil {
		panic(fmt.Sprintf("Failed to setup auth module: %s", err))
	}
	if err := throttler.Setup(&CONF.AdminRedis, logger); err != nil {
		panic(fmt.Sprintf("Failed to setup throttler module: %s", err))
	}
	if err := push.Setup(CONF, logger); err != nil {
		panic(fmt.Sprintf("Failed to setup push module: %s", err))
	}
	handlers.SetupParamDefaults(CONF)
	handlers.Setup(logger)
}

func TestMain(m *testing.M) {
	setup()
	ret := m.Run()
	os.Exit(ret)
}

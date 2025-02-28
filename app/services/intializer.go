package services

import (
	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/saifwork/url-shortner-service/app/configs"
	"github.com/saifwork/url-shortner-service/app/services/domains"
	"github.com/saifwork/url-shortner-service/redisstore"
	"go.mongodb.org/mongo-driver/mongo"
)

type Initializer struct {
	bot        *tgbotapi.BotAPI
	gin        *gin.Engine
	conf       *configs.Config
	redisStore *redisstore.RedisService
	client     *mongo.Client
}

func NewInitializer(bot *tgbotapi.BotAPI, gin *gin.Engine, conf *configs.Config, rs *redisstore.RedisService, cli *mongo.Client) *Initializer {
	s := &Initializer{
		bot:        bot,
		gin:        gin,
		conf:       conf,
		redisStore: rs,
		client:     cli,
	}
	return s
}

func (s *Initializer) RegisterDomains(domains []domains.IDomain) {
	for _, domain := range domains {
		domain.SetupRoutes()
	}
}

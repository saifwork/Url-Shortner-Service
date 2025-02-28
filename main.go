package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/saifwork/url-shortner-service/app/configs"
	"github.com/saifwork/url-shortner-service/app/middleware"
	"github.com/saifwork/url-shortner-service/app/services"
	"github.com/saifwork/url-shortner-service/app/services/domains"
	"github.com/saifwork/url-shortner-service/database"
	"github.com/saifwork/url-shortner-service/redisstore"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	runServer()
}

// service restart

func runServer() {
	// Load the configurations
	log.Println("Loading config ...")
	config := configs.NewConfig("")

	bot, err := tgbotapi.NewBotAPI(config.TelegramBotToken)
	if err != nil {
		log.Print("Error creating bot:", err)
		return
	}

	// Load redis connection
	rs := redisstore.NewRedisService(config)

	log.Println("Parsing environment ...")
	host := config.ServiceHost
	port := config.ServicePort
	if host == "" {
		host = "0.0.0.0"
	}
	if port == "" {
		port = "8080"
	}

	// Database connection
	log.Println("Initialize db ...")
	client, err := database.InitMongo(config)
	if err != nil {
		log.Fatal(err)
	}

	// Close the connection
	defer client.Disconnect(context.Background())

	// Setting routes endpoints
	log.Println("Creating the service ...")
	r := gin.New()

	// Global middleware
	log.Printf("Logging channel: %s to %s", config.LoggingChannel, config.LoggingEndpoint)
	if config.LoggingChannel == "file" {
		logfile, err := os.OpenFile(middleware.GetLogfilePath(config), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			panic(err)
		}
		defer func(logfile *os.File) {
			log.Println("Logfile closed")
			_ = logfile.Close()
		}(logfile)

		r.Use(middleware.DefaultStructuredLogger(config, logfile))
	} else {
		log.Printf("Using default gin logger")
		r.Use(gin.Logger())
	}

	// Recovery middleware
	r.Use(gin.Recovery())

	// Enable CORS middleware
	r.Use(CORSMiddleware())

	// Setup services
	r.GET("/healthcheck", Healthcheck)

	SetupRoutes(bot, r, config, rs, client)

	isHttps, err := strconv.Atoi(os.Getenv("SERVICE_HTTPS"))
	if err == nil && isHttps == 1 {
		crt := os.Getenv("SERVICE_CERT")
		key := os.Getenv("SERVICE_KEY")
		log.Printf("Starting the HTTPS server on %s:%s", host, port)
		err := r.RunTLS(fmt.Sprintf("%s:%s", host, port), crt, key)
		if err != nil {
			log.Fatalf("Error on starting the service: %v", err)
		}
	} else {
		log.Printf("Starting the HTTP server on %s:%s", host, port)
		err := r.Run(fmt.Sprintf("%s:%s", host, port))
		if err != nil {
			log.Fatalf("Error on starting the service: %v", err)
		}
	}
}

func Healthcheck(c *gin.Context) {
	version := os.Getenv("VERSION")
	if version == "" {
		version = "OK"
	}
	response := map[string]string{
		"status":  "up",
		"version": version,
	}
	c.JSON(http.StatusOK, response)
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func SetupRoutes(bot *tgbotapi.BotAPI, gin *gin.Engine, conf *configs.Config, rs *redisstore.RedisService, cli *mongo.Client) {
	initializer := services.NewInitializer(bot, gin, conf, rs, cli)

	var domain []domains.IDomain

	us := domains.NewUrlShortnerService(bot, gin, conf, rs, cli)

	domain = append(domain, us)

	initializer.RegisterDomains(domain)
}

package configs

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Version      string // Version of the application
	ServiceName  string // Name of the service
	ServiceHost  string // Host of the service
	ServicePort  string // Port of the service
	ServiceHTTPS string // HTTPS of the service

	TelegramBotToken string // Telegram bot token

	RedisHost       string
	RedisPwd        string
	RedisPrefix     string
	RedisCounterKey string
	UrlSalt         string

	MongoDSN                    string // MongoDB DSN (Data Source Name)
	MongoDatabase               string // MongoDB database name
	MongoMaxPoolSize            uint64 // MongoDB maximum pool size
	MongoSocketTimeout          uint64 // MongoDB socket timeout (seconds)
	MongoServerSelectionTimeout uint64 // MongoDB server selection timeout (seconds)
	MongoTimeoutSeconds         uint64 // MongoDB timeout (seconds)
	MongoConnectTimeoutSeconds  uint64 // MongoDB connection timeout (seconds)

	LoggingLevel    int    // Logging level (integer value)
	LoggingChannel  string // Logging channel (file, database, etc.)
	LoggingEndpoint string // Logging endpoint (file path, database URL, etc.)
}

func NewConfig(envPath string) *Config {
	c := Config{}
	if envPath == "" {
		envPath = ".env"
	}
	c.initialize(envPath)
	return &c
}

func (c *Config) initialize(envPath string) {
	// Load config
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Environment file missed. Err: %s", err)
	}

	c.Version = os.Getenv("VERSION")
	if c.Version == "" {
		log.Panicln("VERSION not specified")
	}

	c.ServiceName = os.Getenv("SERVICE_NAME")
	if c.ServiceName == "" {
		log.Panicln("SERVICE_NAME not specified")
	}

	c.ServiceHost = os.Getenv("SERVICE_HOST")
	c.ServicePort = os.Getenv("PORT")
	c.ServiceHTTPS = os.Getenv("SERVICE_HTTPS")

	c.TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if c.TelegramBotToken == "" {
		log.Panicln("TELEGRAM_BOT_TOKEN not specified")
	}
	
	c.RedisHost = os.Getenv("REDIS_HOST")
	c.RedisPwd = os.Getenv("REDIS_PWD")
	c.RedisPrefix = os.Getenv("REDIS_KEY_PREFIX")
	c.RedisCounterKey = os.Getenv("REDIS_COUNTER_KEY")
	c.UrlSalt = os.Getenv("URL_SALT")

	c.MongoDSN = os.Getenv("MONGO_DSN")
	c.MongoDatabase = os.Getenv("MONGO_DATABASE")

	c.MongoMaxPoolSize, _ = strconv.ParseUint(os.Getenv("MONGO_MAX_POOL_SIZE"), 10, 64)
	c.MongoSocketTimeout, _ = strconv.ParseUint(os.Getenv("MONGO_SECONDS_SOCKET_TIMEOUT"), 10, 64)
	c.MongoServerSelectionTimeout, _ = strconv.ParseUint(os.Getenv("MONGO_SECONDS_SERVER_SELECTION_TIMEOUT"), 10, 64)
	c.MongoTimeoutSeconds, _ = strconv.ParseUint(os.Getenv("MONGO_SECONDS_TIMEOUT"), 10, 64)
	c.MongoConnectTimeoutSeconds, _ = strconv.ParseUint(os.Getenv("MONGO_SECONDS_CONNECTION_TIMEOUT"), 10, 64)

	ll, err := strconv.Atoi(os.Getenv("LOGGING_LEVEL"))
	if err != nil {
		ll = 3 // Default value
	}
	c.LoggingLevel = ll
	c.LoggingEndpoint = os.Getenv("LOGGING_ENDPOINT")
	c.LoggingChannel = os.Getenv("LOGGING_CHANNEL")
}

// GetBaseURL constructs the base URL dynamically
func (c *Config) GetBaseURL() string {
	return fmt.Sprintf("http://%s:%s", c.ServiceHost, c.ServicePort)
}

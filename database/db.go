package database

import (
	"context"
	"fmt"
	"time"

	"github.com/saifwork/url-shortner-service/app/configs"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func InitMongo(config *configs.Config) (*mongo.Client, error) {
	return ConnectToMongoDB(config)
}

// ConnectToMongoDB connects to MongoDB and returns the client
func ConnectToMongoDB(config *configs.Config) (*mongo.Client, error) {
	// Set client options
	clientOptions := options.Client().
		ApplyURI(config.MongoDSN).
		SetMaxPoolSize(config.MongoMaxPoolSize).
		SetSocketTimeout(time.Duration(config.MongoSocketTimeout) * time.Second).
		SetServerSelectionTimeout(time.Duration(config.MongoServerSelectionTimeout) * time.Second).
		SetConnectTimeout(time.Duration(config.MongoConnectTimeoutSeconds) * time.Second).
		SetAppName(config.ServiceName)

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}

	// Check the connection
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to MongoDB!")

	return client, nil
}

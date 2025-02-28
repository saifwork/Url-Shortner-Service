package dtos

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ShortenedURL struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	ChatID     int64              `bson:"chat_id"`    // Store user’s Telegram Chat ID
	Original   string             `bson:"original"`   // Original long URL
	Shortened  string             `bson:"shortened"`  // Shortened URL
	Clicks     int                `bson:"clicks"`     // Track number of visits
	CreatedAt  time.Time          `bson:"created_at"` // Timestamp
	FirstClick time.Time          `bson:"first_click"`
	LastClick  time.Time          `bson:"last_click"`
	Countries  []string           `bson:"countries"` // ["India", "USA"]
	Cities     []string           `bson:"cities"`    // ["Mumbai", "New York"]
	Devices    []string           `bson:"devices"`   // ["Mobile", "Desktop"]
	OS         []string           `bson:"os"`        // ["Windows", "Android"]
	Browsers   []string           `bson:"browsers"`  // ["Chrome", "Firefox"]
}

type Feedback struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	ChatID    int64              `bson:"chat_id"`
	Message   string             `bson:"message"`
	Timestamp time.Time          `bson:"timestamp"`
}

type IPInfo struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
	Query       string  `json:"query"`
}

package domains

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/mileusna/useragent"
	"github.com/saifwork/url-shortner-service/app/configs"
	"github.com/saifwork/url-shortner-service/app/services/core/responses"
	"github.com/saifwork/url-shortner-service/app/services/domains/dtos"
	"github.com/saifwork/url-shortner-service/app/utils"
	"github.com/saifwork/url-shortner-service/redisstore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UrlShortnerService struct {
	Bot        *tgbotapi.BotAPI
	Gin        *gin.Engine
	Conf       *configs.Config
	redisStore *redisstore.RedisService
	Client     *mongo.Client
}

func NewUrlShortnerService(bot *tgbotapi.BotAPI, gin *gin.Engine, conf *configs.Config, rs *redisstore.RedisService, cli *mongo.Client) *UrlShortnerService {
	return &UrlShortnerService{
		Bot:        bot,
		Gin:        gin,
		Conf:       conf,
		redisStore: rs,
		Client:     cli,
	}
}

func (s *UrlShortnerService) SetupRoutes() {
	g := s.Gin.Group("")

	g.GET("/:shortID", s.RedirectShortURL) // List tracked products

	go s.StartConsuming()
}

func (s *UrlShortnerService) StartConsuming() {
	// Start consuming

	fmt.Println("✅ Bot is running...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, _ := s.Bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		switch {
		case text == "/start":
			s.HandleStart(chatID)

		case text == "/help":
			s.HandleHelp(chatID)

		case strings.HasPrefix(text, "/shorten "):
			s.HandleShorten(chatID, strings.TrimPrefix(text, "/shorten "))

		case text == "/my_links":
			s.HandleMyLinks(chatID)

		case strings.HasPrefix(text, "/stats "):
			s.HandleStats(chatID, strings.TrimPrefix(text, "/stats "))

		case strings.HasPrefix(text, "/delete "):
			s.HandleDelete(chatID, strings.TrimPrefix(text, "/delete "))

		case strings.HasPrefix(text, "/feedback "):
			s.HandleFeedback(chatID, strings.TrimPrefix(text, "/feedback "))

		case text == "/about":
			s.HandleAbout(chatID)

		default:
			s.HandleUnknownCommand(chatID)
		}
	}
}

func (s *UrlShortnerService) HandleStart(chatID int64) {
	msg := "👋 Welcome to the URL Shortener Bot! 🚀\n\n" +
		"🔹 Use `/shorten <URL>` to shorten a link.\n" +
		"🔹 Use `/my_links` to view your shortened links.\n" +
		"🔹 Use `/stats <short_url>` to see click stats.\n" +
		"🔹 Use `/delete <short_url>` to remove a link.\n" +
		"🔹 Use `/feedback <message>` to share feedback.\n" +
		"🔹 Use `/about` to learn more.\n\n" +
		"Type `/help` for detailed instructions."

	s.Bot.Send(tgbotapi.NewMessage(chatID, msg))
}

func (s *UrlShortnerService) HandleHelp(chatID int64) {
	msg := "📖 **How to Use the URL Shortener Bot:**\n\n" +
		"✅ **Shorten a URL:** `/shorten https://example.com`\n" +
		"✅ **View your links:** `/my_links`\n" +
		"✅ **Check stats:** `/stats <short_url>`\n" +
		"✅ **Delete a link:** `/delete <short_url>`\n" +
		"✅ **Give feedback:** `/feedback I love this bot!`\n" +
		"✅ **Learn about the bot:** `/about`\n\n" +
		"🔹 Use `/start` to see the welcome message again.\n" +
		"🔹 Have questions? Just send a message!"

	s.Bot.Send(tgbotapi.NewMessage(chatID, msg))
}

func (s *UrlShortnerService) HandleUnknownCommand(chatID int64) {
	msg := "❌ Unknown command. Type `/help` to see available commands."
	s.Bot.Send(tgbotapi.NewMessage(chatID, msg))
}

func (s *UrlShortnerService) HandleShorten(chatID int64, url string) {
	if url == "" {
		msg := tgbotapi.NewMessage(chatID, "Please provide a valid URL. Example: `/shorten https://example.com`")
		s.Bot.Send(msg)
		return
	}

	shortURL, err := s.GenerateShortURL(chatID, url)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Failed to shorten URL. Please try again.")
		s.Bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Shortened URL: %s", shortURL))
	s.Bot.Send(msg)
}

// Function to generate a short URL and save it in DB
func (s *UrlShortnerService) GenerateShortURL(chatID int64, originalURL string) (string, error) {
	shortID, err := utils.GenerateShortID(s.redisStore)
	if err != nil {
		return "", err
	}

	log.Println("Generated Short ID:", shortID)

	shortURL := fmt.Sprintf("http://localhost:8080/%s", shortID)

	// Store in DB
	newURL := dtos.ShortenedURL{
		ChatID:     chatID,
		Original:   originalURL,
		Shortened:  shortURL,
		Clicks:     0,
		CreatedAt:  time.Now(),
		FirstClick: time.Time{}, // Empty until first click
		LastClick:  time.Time{}, // Empty until last click
		Countries:  []string{},  // Initialize empty slice
		Cities:     []string{},  // Initialize empty slice
		Devices:    []string{},  // Initialize empty slice
		OS:         []string{},  // Initialize empty slice
		Browsers:   []string{},  // Initialize empty slice
	}

	_, err = s.Client.Database(s.Conf.MongoDatabase).Collection("urls").InsertOne(context.Background(), newURL)
	if err != nil {
		return "", err
	}

	return shortURL, nil
}

func (s *UrlShortnerService) HandleMyLinks(chatID int64) {
	var results []dtos.ShortenedURL

	cursor, err := s.Client.Database(s.Conf.MongoDatabase).Collection("urls").Find(context.TODO(), bson.M{"chat_id": chatID})
	if err != nil {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Error fetching your links. Please try again."))
		return
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var link dtos.ShortenedURL
		cursor.Decode(&link)
		results = append(results, link)
	}

	if len(results) == 0 {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "You haven't shortened any URLs yet. Use `/shorten <URL>` to start."))
		return
	}

	msg := "📌 **Your Shortened Links:**\n"
	for _, link := range results {
		msg += fmt.Sprintf("🔹 [%s](%s) → %s\n", link.Shortened, link.Shortened, link.Original)
	}

	s.Bot.Send(tgbotapi.NewMessage(chatID, msg))
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "None"
	}
	return strings.Join(items, ", ")
}

func (s *UrlShortnerService) HandleStats(chatID int64, shortURL string) {
	if shortURL == "" {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "Please provide a shortened URL. Example: `/stats https://yourshort.ly/abc123`"))
		return
	}

	var result dtos.ShortenedURL
	err := s.Client.Database(s.Conf.MongoDatabase).Collection("urls").
		FindOne(context.TODO(), bson.M{"shortened": shortURL}).
		Decode(&result)

	if err != nil {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "❌ URL not found. Please check and try again."))
		return
	}

	// Ensure only the creator (original user) can view stats
	if result.ChatID != chatID {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "❌ You are not authorized to view stats for this URL."))
		return
	}

	// Convert timestamps to readable format
	firstClick := "Not yet clicked"
	lastClick := "Not yet clicked"
	if !result.FirstClick.IsZero() {
		firstClick = result.FirstClick.Format("02 Jan 2006, 15:04")
	}
	if !result.LastClick.IsZero() {
		lastClick = result.LastClick.Format("02 Jan 2006, 15:04")
	}

	// Format stats message
	msg := fmt.Sprintf(
		"📊 **URL Stats:**\n"+
			"🔗 Short URL: [%s](%s)\n"+
			"📥 Total Clicks: %d\n"+
			"📅 Created: %s\n"+
			"🚀 First Click: %s\n"+
			"⏳ Last Click: %s\n\n"+
			"🌍 **Countries:** %s\n"+
			"🏙️ **Cities:** %s\n"+
			"📱 **Devices:** %s\n"+
			"💻 **OS:** %s\n"+
			"🌐 **Browsers:** %s",
		result.Shortened, result.Shortened, result.Clicks,
		result.CreatedAt.Format("02 Jan 2006, 15:04"),
		firstClick, lastClick,
		formatList(result.Countries),
		formatList(result.Cities),
		formatList(result.Devices),
		formatList(result.OS),
		formatList(result.Browsers),
	)

	s.Bot.Send(tgbotapi.NewMessage(chatID, msg))
}

func (s *UrlShortnerService) HandleDelete(chatID int64, shortURL string) {
	if shortURL == "" {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "Please provide a shortened URL to delete. Example: `/delete https://yourshort.ly/abc123`"))
		return
	}

	// Find and delete the URL from DB
	res, err := s.Client.Database(s.Conf.MongoDatabase).Collection("urls").DeleteOne(context.TODO(), bson.M{"shortened": shortURL, "chat_id": chatID})
	if err != nil || res.DeletedCount == 0 {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Unable to delete URL. Make sure it's yours and try again."))
		return
	}

	s.Bot.Send(tgbotapi.NewMessage(chatID, "✅ URL deleted successfully."))
}

func (s *UrlShortnerService) HandleAbout(chatID int64) {
	msg := "🚀 **URL Shortener Bot**\n" +
		"🔹 Shorten your long links.\n" +
		"🔹 Track analytics and clicks.\n" +
		"🔹 Manage your links easily.\n\n" +
		"💡 Use `/help` to see available commands!"

	s.Bot.Send(tgbotapi.NewMessage(chatID, msg))
}

func (s *UrlShortnerService) HandleFeedback(chatID int64, feedback string) {
	if feedback == "" {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "Please provide your feedback. Example: `/feedback I love this bot!`"))
		return
	}

	// Check if user has given feedback in the last 7 days
	var lastFeedback dtos.Feedback
	err := s.Client.Database(s.Conf.MongoDatabase).Collection("feedbacks").FindOne(
		context.TODO(),
		bson.M{"chat_id": chatID},
		options.FindOne().SetSort(bson.M{"timestamp": -1}),
	).Decode(&lastFeedback)

	if err == nil {
		// Calculate time difference
		oneWeekAgo := time.Now().AddDate(0, 0, -7)
		if lastFeedback.Timestamp.After(oneWeekAgo) {
			s.Bot.Send(tgbotapi.NewMessage(chatID, "❌ You can only provide feedback once a week. Try again later!"))
			return
		}
	}

	// Create a new feedback entry
	newFeedback := dtos.Feedback{
		ChatID:    chatID,
		Message:   feedback,
		Timestamp: time.Now(),
	}

	// Insert into MongoDB
	_, err = s.Client.Database(s.Conf.MongoDatabase).Collection("feedbacks").InsertOne(context.TODO(), newFeedback)
	if err != nil {
		s.Bot.Send(tgbotapi.NewMessage(chatID, "❌ Failed to save feedback. Please try again later."))
		return
	}

	// Confirmation message
	s.Bot.Send(tgbotapi.NewMessage(chatID, "✅ Thank you for your feedback! 😊"))

	fmt.Printf("Feedback saved: %+v\n", newFeedback)
}

func (s *UrlShortnerService) RedirectShortURL(c *gin.Context) {
	shortID := c.Param("shortID")
	ctx := context.Background()

	// Get the real client IP
	ip := c.ClientIP()
	log.Printf("Client IP: %s\n", ip)

	// Try fetching from Redis first
	longURL, err := s.redisStore.Get(shortID)
	if err == nil && longURL != "" {
		// Redirect immediately
		c.Redirect(http.StatusFound, longURL)

		// Process stats asynchronously
		go s.ProcessClickStats(shortID, c)

		return
	}

	// If not found in Redis, check MongoDB
	var result dtos.ShortenedURL
	err = s.Client.Database(s.Conf.MongoDatabase).Collection("urls").
		FindOne(ctx, bson.M{"shortened": "http://localhost:8080/" + shortID}).
		Decode(&result)

	if err != nil {
		log.Println("Short URL not found in MongoDB:", err)
		c.AbortWithStatusJSON(http.StatusNotFound, responses.NewErrorResponse(http.StatusNotFound, "Short URL not found", nil))
		return
	}

	// Redirect immediately
	c.Redirect(http.StatusFound, result.Original)

	// Process stats asynchronously
	go s.ProcessClickStats(shortID, c)

	// Cache the long URL in Redis for faster future lookups
	go func() {
		cacheErr := s.redisStore.Set(shortID, result.Original, 24*time.Hour)
		if cacheErr != nil {
			log.Println("Error caching URL in Redis:", cacheErr)
		}
	}()
}

func (s *UrlShortnerService) ProcessClickStats(shortID string, c *gin.Context) {
	// Fetch user metadata
	ip := c.ClientIP()

	log.Println("IP Address:", ip)
	userAgent := c.GetHeader("User-Agent")
	country, city, device, os, browser := s.ExtractUserMetadata(ip, userAgent)

	// Update click stats in MongoDB
	err := s.UpdateClickStats(shortID, country, city, device, os, browser)
	if err != nil {
		log.Println("Error updating click stats:", err)
	}
}

func (s *UrlShortnerService) UpdateClickStats(shortID, country, city, device, os, browser string) error {
	ctx := context.Background()
	collection := s.Client.Database(s.Conf.MongoDatabase).Collection("urls")

	// MongoDB update query
	update := bson.M{
		"$addToSet": bson.M{ // Ensures only unique values are added
			"countries": country,
			"cities":    city,
			"devices":   device,
			"os":        os,
			"browsers":  browser,
		},
		"$inc": bson.M{ // Increment total clicks
			"clicks": 1,
		},
		"$set": bson.M{ // Update last click timestamp
			"last_click": time.Now(),
		},
	}

	// Fetch existing document to check if first click is set
	filter := bson.M{"shortened": "http://localhost:8080/" + shortID}
	var result dtos.ShortenedURL
	err := collection.FindOne(ctx, filter).Decode(&result)
	if err == nil && result.FirstClick.IsZero() {
		update["$set"].(bson.M)["first_click"] = time.Now()
	}

	_, err = collection.UpdateOne(ctx, filter, update)
	return err
}

func (s *UrlShortnerService) ExtractUserMetadata(ip, userAgent string) (string, string, string, string, string) {
	// Example: Fetch country & city from an IP lookup API (replace with real implementation)
	country := "Unknown"
	city := "Unknown"

	ipInfo, err := GetIPInfo(ip)
	if err != nil {
		log.Fatal(err)
	}
	country = ipInfo.Country
	city = ipInfo.City
	fmt.Printf("Country: %s, City: %s, ISP: %s\n", ipInfo.Country, ipInfo.City, ipInfo.ISP)

	// Parse User-Agent to get device, OS, and browser
	ua := useragent.Parse(userAgent)

	// Detect Device Type
	var device string
	switch {
	case ua.IsAndroid() || ua.IsIOS():
		device = "Mobile"
	case ua.IsMacOS() || ua.IsWindows() || ua.IsLinux():
		device = "Desktop"
	default:
		device = "Unknown"
	}

	// Detect Operating System
	var os string
	switch {
	case ua.IsWindows():
		os = "Windows"
	case ua.IsMacOS():
		os = "MacOS"
	case ua.IsLinux():
		os = "Linux"
	case ua.IsAndroid():
		os = "Android"
	case ua.IsIOS():
		os = "iOS"
	default:
		os = "Unknown"
	}

	// Detect Browser
	var browser string
	switch {
	case ua.IsChrome():
		browser = "Chrome"
	case ua.IsFirefox():
		browser = "Firefox"
	case ua.IsSafari():
		browser = "Safari"
	case ua.IsEdge():
		browser = "Edge"
	case ua.IsOpera():
		browser = "Opera"
	case ua.IsInternetExplorer():
		browser = "Internet Explorer"
	default:
		browser = "Unknown"
	}

	return country, city, device, os, browser
}

func GetIPInfo(ip string) (*dtos.IPInfo, error) {
	url := fmt.Sprintf("http://ip-api.com/json/%s", ip)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	log.Println("resp:", resp)

	var ipInfo dtos.IPInfo
	err = json.NewDecoder(resp.Body).Decode(&ipInfo)
	if err != nil {
		return nil, err
	}

	return &ipInfo, nil
}

package utils

import (
	"context"
	"log"

	"github.com/saifwork/url-shortner-service/redisstore"
	"github.com/speps/go-hashids"
)

const base62Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func GenerateShortID(rs *redisstore.RedisService) (string, error) {
	ctx := context.Background()

	// Increment global counter atomatically in Redis
	id, err := rs.Client.Incr(ctx, rs.Config.RedisCounterKey).Result()
	if err != nil {
		return "", err
	}

	// Create a new hashids instance with the custom alphabet
	hd := hashids.NewData()
	hd.Salt = rs.Config.UrlSalt
	hd.Alphabet = base62Alphabet
	hd.MinLength = 7

	h, err := hashids.NewWithData(hd)
	if err != nil {
		log.Println("Error creating Hashids instance:", err)
		return "", err
	}

	// Convert the counter (id) to a short Base62 ID
	encoded, err := h.Encode([]int{int(id)})
	if err != nil {
		log.Println("Error encoding ID:", err)
		return "", err
	}

	return encoded, nil
}

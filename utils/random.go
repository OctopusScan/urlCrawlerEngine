package utils

import (
	"github.com/google/uuid"
	"math/rand"
)

func GenerateRandomString(length int) string {
	uuidObj, _ := uuid.NewUUID()
	uuidString := uuidObj.String()
	var seed int64
	for _, c := range uuidString {
		seed += int64(c)
	}
	rand.Seed(seed)
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

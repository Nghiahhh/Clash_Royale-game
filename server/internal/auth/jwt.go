package auth

import (
	"context"
	"errors"
	"time"

	"server/internal/config"
	"server/internal/db"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
)

var jwtKey = []byte(config.Config.JWTSecret)

type Claims struct {
	ID       int    `json:"id"`
	Gmail    string `json:"gmail"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken tạo JWT token không có thời hạn và lưu vào MongoDB (tạo bản ghi mới)
func GenerateToken(id int, gmail, username string) (string, error) {
	claims := &Claims{
		ID:       id,
		Gmail:    gmail,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			// Không có ExpiresAt => token vĩnh viễn
		},
	}

	// Tạo chuỗi token
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	// Lưu bản ghi mới vào MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoDatabase.Collection("user_tokens") // tạo riêng collection nếu cần
	doc := bson.M{
		"id":        id,
		"gmail":     gmail,
		"username":  username,
		"token":     token,
		"active_at": time.Now(),
	}

	_, err = collection.InsertOne(ctx, doc)
	if err != nil {
		return "", err
	}

	return token, nil
}

func ValidateTokenWithMongo(tokenStr string) (*Claims, error) {
	// 1. Giải mã token bằng JWT
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	// 2. Truy vấn MongoDB để kiểm tra token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := db.MongoDatabase.Collection("user_tokens")

	filter := bson.M{
		"id":    claims.ID,
		"token": tokenStr,
	}

	var result struct {
		Gmail    string `bson:"gmail"`
		Username string `bson:"username"`
	}

	err = collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, errors.New("token not found in database")
	}

	// 3. Kiểm tra thông tin có khớp với token không
	if result.Gmail != claims.Gmail || result.Username != claims.Username {
		return nil, errors.New("token data mismatch")
	}

	// 4. Cập nhật trường active_at
	update := bson.M{
		"$set": bson.M{
			"active_at": time.Now(),
		},
	}
	_, _ = collection.UpdateOne(ctx, filter, update)

	// 5. Trả về toàn bộ claims
	return claims, nil
}

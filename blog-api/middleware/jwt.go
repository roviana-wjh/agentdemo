package middleware

import (
	"awesomeProject1/model"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	Username string `json:"username"`
	UUID     int    `json:"uuid"`
	jwt.RegisteredClaims
}

func GenerateToken(username string, userId int) (string, error) {
	expire := time.Now().Add(5 * time.Minute)
	claims := &CustomClaims{
		username,
		userId,
		jwt.RegisteredClaims{
			Issuer: "wjh",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("secret"))
	if err != nil {
		return "", err
	}
	j := model.JWts{Token: tokenString, ExpireAt: expire}
	model.DB.Create(&j)
	return tokenString, nil
}

func ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func JwtAuthMiddleware(c *gin.Context) {
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusOK, gin.H{
			"state": "fail",
			"msg":   "无效token",
		})
		c.Abort()
		return
	}
	mc, err := ParseToken(authHeader)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"msg": "无效token",
		})
		c.Abort()
		return
	}
	var jwts model.JWts
	if err := model.DB.Where("token = ?", authHeader).First(&jwts).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"msg": "token 不存在",
		})
		c.Abort()
		return
	}
	if jwts.Token != "" {
		if jwts.ExpireAt.After(time.Now()) {
			jwts.ExpireAt = time.Now().Add(60 * time.Minute)
			model.DB.Save(&jwts)
		} else {
			model.DB.Unscoped().Delete(&jwts)
			c.JSON(http.StatusUnauthorized, gin.H{
				"msg": "token已过期",
			})
			c.Abort()
			return
		}
	} else {
		c.JSON(http.StatusOK, gin.H{
			"state": "fail",
			"msg":   "无效token",
		})
		c.Abort()
		return
	}
	c.Set("userId", uint(mc.UUID))
	c.Set("username", mc.Username)
	c.Next()
}

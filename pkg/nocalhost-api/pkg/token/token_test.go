package token

import (
	"github.com/golang-jwt/jwt/v4"
	"testing"
	"time"
)

func TestTokenInvalid(t *testing.T) {
	secret := "jwt_secret"

	// The token content.
	// iss: （Issuer）
	// iat: （Issued At）
	// exp: （Expiration Time）
	// aud: （Audience）
	// sub: （Subject）
	// nbf: （Not Before）
	// jti: （JWT ID）
	tk := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  "1",
		"username": "Anur",
		"uuid":     "UUID",
		"email":    "anur@nocalhost.com",
		"is_admin": "1",
		"nbf":      time.Now().Unix(),
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Unix(),
	})

	// Sign the token with the specified secret.
	tokenString, _ := tk.SignedString([]byte(secret))

	// Parse the token.
	token, err := jwt.Parse(tokenString, secretFunc(secret))

	time.Sleep(time.Second * 10)

	if err != nil {
		println(err.Error())
	}

	// Parse error.
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		println(claims["user_id"].(string))
		println(claims["username"].(string))
		println(claims["uuid"].(string))
		println(claims["email"].(string))
		println(claims["is_admin"].(string))
	} else {
		// the token should be invalid
		println("invalid")
	}
}

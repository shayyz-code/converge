package chat

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type tokenClaims struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Name        string `json:"name"`
	jwt.RegisteredClaims
}

func (h *Hub) authenticateToken(token string) (string, string, error) {
	if h.options.JWTSecret == "" {
		return "", "", errors.New("jwt secret not configured")
	}
	if token == "" {
		return "", "", errors.New("jwt token missing")
	}
	claims := &tokenClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(h.options.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return "", "", errors.New("invalid jwt")
	}
	if h.options.JWTIssuer != "" && claims.Issuer != h.options.JWTIssuer {
		return "", "", errors.New("invalid issuer")
	}
	if h.options.JWTAudience != "" && !audienceContains(claims.Audience, h.options.JWTAudience) {
		return "", "", errors.New("invalid audience")
	}
	userID := claims.UserID
	if userID == "" {
		userID = claims.Subject
	}
	if userID == "" {
		return "", "", errors.New("user id missing")
	}
	displayName := strings.TrimSpace(claims.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(claims.Name)
	}
	if displayName == "" {
		displayName = userID
	}
	return userID, displayName, nil
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	query := r.URL.Query()
	if token := query.Get("token"); token != "" {
		return token
	}
	if token := query.Get("access_token"); token != "" {
		return token
	}
	return ""
}

func audienceContains(audience jwt.ClaimStrings, value string) bool {
	for _, aud := range audience {
		if aud == value {
			return true
		}
	}
	return false
}

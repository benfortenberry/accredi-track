package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
)

// AuthMiddleware validates the JWT
func AuthMiddleware() gin.HandlerFunc {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	return func(c *gin.Context) {
		// Get Auth0 domain and audience from environment variables
		auth0Domain := os.Getenv("AUTH0_DOMAIN")
		auth0Audience := os.Getenv("AUTH0_AUDIENCE")
		if auth0Domain == "" || auth0Audience == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Auth0 configuration is missing"})
			c.Abort()
			return
		}

		// Fetch JWKS from Auth0
		jwksURL := fmt.Sprintf("https://%s/.well-known/jwks.json", auth0Domain)
		jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch JWKS"})
			c.Abort()
			return
		}

		// Extract the token from the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix from the token
		tokenString := authHeader[len("Bearer "):]

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, jwks.Keyfunc)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Validate claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

			sub, ok := claims["sub"].(string)
			if !ok {
				fmt.Println("Error: sub claim is missing")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: missing sub claim"})
				c.Abort()
				return
			}
			// fmt.Println(sub)
			c.Set("userSub", sub)

			// Validate audience
			audClaim, ok := claims["aud"]
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Audience claim is missing"})
				c.Abort()
				return
			}

			// Handle single or multiple audiences
			switch aud := audClaim.(type) {
			case string:
				// Single audience
				if aud != auth0Audience {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid audience"})
					c.Abort()
					return
				}
			case []interface{}:
				// Multiple audiences
				validAudience := false
				for _, a := range aud {
					if aStr, ok := a.(string); ok {
						if strings.TrimSpace(aStr) == strings.TrimSpace(auth0Audience) {
							validAudience = true
							break
						}
					}
				}
				if !validAudience {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid audience"})
					c.Abort()
					return
				}
			default:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid audience format"})
				c.Abort()
				return
			}

			// Validate issuer
			expectedIssuer := fmt.Sprintf("https://%s/", auth0Domain)
			if claims["iss"] != expectedIssuer {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid issuer"})
				c.Abort()
				return
			}

			// Token is valid, proceed to the next handler
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
		}
	}
}

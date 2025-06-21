package api

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stavily/agents/shared/pkg/config"
	"go.uber.org/zap"
)

// AuthManager handles authentication for API requests
type AuthManager struct {
	config config.AuthConfig
	logger *zap.Logger

	// JWT-specific fields
	jwtSecret  []byte
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey

	// Token management
	currentToken string
	tokenExpiry  time.Time
	mu           sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config config.AuthConfig, logger *zap.Logger) (*AuthManager, error) {
	manager := &AuthManager{
		config: config,
		logger: logger,
	}

	switch config.Method {
	case "jwt":
		if err := manager.initJWT(); err != nil {
			return nil, fmt.Errorf("failed to initialize JWT auth: %w", err)
		}
	case "certificate":
		// Certificate-based auth is handled by TLS configuration
		// No additional setup needed here
	default:
		return nil, fmt.Errorf("unsupported authentication method: %s", config.Method)
	}

	return manager, nil
}

// initJWT initializes JWT authentication
func (a *AuthManager) initJWT() error {
	if a.config.JWT.SecretFile != "" {
		secret, err := os.ReadFile(a.config.JWT.SecretFile)
		if err != nil {
			return fmt.Errorf("failed to read JWT secret file: %w", err)
		}

		// Check if it's a PEM-encoded key
		if block, _ := pem.Decode(secret); block != nil {
			switch block.Type {
			case "RSA PRIVATE KEY":
				privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
				if err != nil {
					return fmt.Errorf("failed to parse RSA private key: %w", err)
				}
				a.privateKey = privateKey
				a.publicKey = &privateKey.PublicKey

			case "PUBLIC KEY":
				publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
				if err != nil {
					return fmt.Errorf("failed to parse public key: %w", err)
				}
				rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
				if !ok {
					return fmt.Errorf("public key is not RSA")
				}
				a.publicKey = rsaPublicKey

			default:
				return fmt.Errorf("unsupported key type: %s", block.Type)
			}
		} else {
			// Treat as raw secret for HMAC
			a.jwtSecret = secret
		}
	}

	return nil
}

// AddAuth adds authentication to an HTTP request
func (a *AuthManager) AddAuth(req *http.Request) error {
	switch a.config.Method {
	case "jwt":
		return a.addJWTAuth(req)
	case "certificate":
		// Certificate auth is handled by TLS, nothing to add to request
		return nil
	default:
		return fmt.Errorf("unsupported authentication method: %s", a.config.Method)
	}
}

// addJWTAuth adds JWT authentication to the request
func (a *AuthManager) addJWTAuth(req *http.Request) error {
	token, err := a.getValidToken()
	if err != nil {
		return fmt.Errorf("failed to get valid token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// getValidToken returns a valid JWT token, creating a new one if necessary
func (a *AuthManager) getValidToken() (string, error) {
	a.mu.RLock()
	if a.currentToken != "" && time.Now().Before(a.tokenExpiry.Add(-time.Minute)) {
		token := a.currentToken
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after acquiring write lock
	if a.currentToken != "" && time.Now().Before(a.tokenExpiry.Add(-time.Minute)) {
		return a.currentToken, nil
	}

	// Create new token
	token, expiry, err := a.createJWTToken()
	if err != nil {
		return "", fmt.Errorf("failed to create JWT token: %w", err)
	}

	a.currentToken = token
	a.tokenExpiry = expiry

	return token, nil
}

// createJWTToken creates a new JWT token
func (a *AuthManager) createJWTToken() (string, time.Time, error) {
	now := time.Now()
	expiry := now.Add(a.config.TokenTTL)

	claims := &jwt.RegisteredClaims{
		Issuer:    a.config.JWT.Issuer,
		Audience:  jwt.ClaimStrings{a.config.JWT.Audience},
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiry),
	}

	var token *jwt.Token
	var signingKey interface{}

	switch a.config.JWT.Algorithm {
	case "HS256", "HS384", "HS512":
		token = jwt.NewWithClaims(jwt.GetSigningMethod(a.config.JWT.Algorithm), claims)
		signingKey = a.jwtSecret

	case "RS256", "RS384", "RS512":
		token = jwt.NewWithClaims(jwt.GetSigningMethod(a.config.JWT.Algorithm), claims)
		signingKey = a.privateKey

	default:
		return "", time.Time{}, fmt.Errorf("unsupported JWT algorithm: %s", a.config.JWT.Algorithm)
	}

	if signingKey == nil {
		return "", time.Time{}, fmt.Errorf("no signing key available for algorithm %s", a.config.JWT.Algorithm)
	}

	tokenString, err := token.SignedString(signingKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign JWT token: %w", err)
	}

	a.logger.Debug("Created new JWT token",
		zap.String("algorithm", a.config.JWT.Algorithm),
		zap.Time("expiry", expiry))

	return tokenString, expiry, nil
}

// RefreshToken refreshes the current token if supported
func (a *AuthManager) RefreshToken() error {
	if a.config.Method != "jwt" || !a.config.JWT.RefreshToken {
		return fmt.Errorf("token refresh not supported for auth method: %s", a.config.Method)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Clear current token to force creation of new one
	a.currentToken = ""
	a.tokenExpiry = time.Time{}

	a.logger.Debug("Token refresh requested")

	return nil
}

// ValidateToken validates a JWT token
func (a *AuthManager) ValidateToken(tokenString string) (*jwt.RegisteredClaims, error) {
	if a.config.Method != "jwt" {
		return nil, fmt.Errorf("token validation not supported for auth method: %s", a.config.Method)
	}

	var keyFunc jwt.Keyfunc

	switch a.config.JWT.Algorithm {
	case "HS256", "HS384", "HS512":
		keyFunc = func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return a.jwtSecret, nil
		}

	case "RS256", "RS384", "RS512":
		keyFunc = func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return a.publicKey, nil
		}

	default:
		return nil, fmt.Errorf("unsupported JWT algorithm: %s", a.config.JWT.Algorithm)
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, keyFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid JWT token")
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, fmt.Errorf("invalid JWT claims")
	}

	return claims, nil
}

// GetTokenInfo returns information about the current token
func (a *AuthManager) GetTokenInfo() (string, time.Time, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	hasToken := a.currentToken != ""
	return a.currentToken, a.tokenExpiry, hasToken
}

// Close cleans up the authentication manager
func (a *AuthManager) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Clear sensitive data
	a.currentToken = ""
	a.jwtSecret = nil
	a.privateKey = nil
	a.publicKey = nil

	return nil
}

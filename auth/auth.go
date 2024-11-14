package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
	"log/slog"
	"os"
	"time"
)

// TokenIndex defines an interface for managing tokens and their associated users.
// It provides methods to find, insert, update, and remove tokens.
type TokenIndex[token string, user Session] interface {
	Find(t token) (u user, found bool)
	Upsert(t token, check index_utils.UpdateCheck[string, Session]) (updated bool, err error)
	Remove(t token) (removedUser user, removed bool)
}

// UserIndex defines an interface for managing users and their associated tokens.
// It provides methods to find, insert, update, and remove users.
type UserIndex[user string, token Session] interface {
	Find(t user) (u token, found bool)
	Upsert(t user, check index_utils.UpdateCheck[string, Session]) (updated bool, err error)
	Remove(t user) (removedToken token, removed bool)
}

// Session holds the token and expiration information for an authenticated user.
type Session struct {
	user       string // Bearer token
	expiration int64  // Unix timestamp for expiration
}

// AuthStruct is responsible for managing user sessions and their corresponding tokens.
// It contains indexes to map users to tokens and vice versa.
type AuthStruct struct {
	// Map of users to tokens
	tokenToUser TokenIndex[string, Session] // Map of tokens to users
}

// New initializes a new AuthStruct by loading existing tokens from a file.
// If the token file is malformed or cannot be loaded, it returns a new AuthStruct
// with empty token and user mappings.
func New(tokenToUserMap TokenIndex[string, Session], tokenFile string) *AuthStruct {

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		slog.Warn("Token file failed to load")
		return &AuthStruct{tokenToUser: tokenToUserMap}
	}

	tokens := make(map[string]string)
	err = json.Unmarshal(data, &tokens)
	if err != nil { //loading tokens
		slog.Warn("Malformed token json detected")
		return &AuthStruct{tokenToUser: tokenToUserMap}
	}
	for user, token := range tokens {
		tokenToUserMap.Upsert(token, func(string, Session, bool) (Session, error) {
			return Session{user: user, expiration: time.Now().AddDate(0, 0, 1).Unix()}, nil
		})

	}

	return &AuthStruct{tokenToUser: tokenToUserMap}
}

// CreateSession generates a new session for the specified username.
// If the username is empty or the user already has a valid session, it returns an error.
// A randomly generated token is created with a 1-hour expiration time.
func (a *AuthStruct) CreateSession(username string) (string, error) {

	// Ensure the username is not empty
	if username == "" {
		slog.Error("Failed to create session: username is empty")
		return "", errors.New("username is empty")
	}

	// Generate a random token
	token, err := randomGeneratedToken()
	if err != nil {
		slog.Error("Error generating token", slog.String("user", username), slog.String("error", err.Error()))
		return "", errors.New("failed to generate token")
	}

	// Store the token and expiration in the users map with the user as the key

	check2 := func(key string, val Session, exists bool) (newVal Session, err error) {
		return Session{user: username, expiration: time.Now().Add(time.Hour).Unix()}, nil
	}
	a.tokenToUser.Upsert(token, check2)
	// Log session creation
	slog.Info("Session created", slog.String("user",
		username), slog.String("token", token))

	// Return the token and no error
	return token, nil
}

// ValidateSession checks if the provided token is valid and still active.
// It returns the associated username if the session is valid, otherwise it returns an error.
func (a *AuthStruct) ValidateSession(token string) (string, error) {
	// Search for the token in all userToToken
	var sessionValid bool

	user, found := a.tokenToUser.Find(token)

	if !found {
		return "", fmt.Errorf("Missing or invalid bearer token")
	}

	sessionValid = time.Now().Unix() < user.expiration

	if !sessionValid {
		slog.Error("Session validation failed: token not found", slog.String("token", token))
		return "", fmt.Errorf("Missing or invalid bearer token")
	}
	slog.Debug(fmt.Sprintf("User retrieved: %s", user.user))
	// Session is valid
	slog.Info("Session is valid", slog.String("token", token))
	return user.user, nil
}

// Login creates or refreshes a session for the specified user.
// If a session exists, it is refreshed with a new expiration time. A new token is generated otherwise.
func (a *AuthStruct) Login(username string) (string, error) {

	token, err := randomGeneratedToken()

	userCheck := func(key string, curVal Session, exists bool) (newVal Session, err error) {

		return Session{user: username, expiration: time.Now().Add(time.Hour).Unix()}, nil
	}

	_, err = a.tokenToUser.Upsert(token, userCheck)

	if err != nil {
		return "", err
	}
	slog.Info("Session created", slog.String("user", username), slog.String("token", token))
	return token, nil
}

// A function to logout a session
func (a *AuthStruct) Logout(token string) (bool, error) {

	// Find the user associated with the token
	_, removed := a.tokenToUser.Remove(token)

	if !removed {
		return false, fmt.Errorf("missing or invalid bearer token")
	}

	slog.Info("Session logged out", slog.String("token", token))
	return true, nil
}

// randomGeneratedToken generates a cryptographically secure random token
// and encodes it as a URL-safe base64 string.
func randomGeneratedToken() (string, error) {
	// Generate 16 random bytes (128 bits)
	byteLength := 14 // 16 bytes * 8 bits/byte = 128 bits
	bytes := make([]byte, byteLength)

	// Read random bytes from crypto/rand (in standard library)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode the bytes into a safe URL string
	bearer_token := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)

	// Return
	return bearer_token, nil
}

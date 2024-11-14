package auth

import (
	"fmt"
	"testing"

	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"

	"github.com/RICE-COMP318-FALL24/owldb-p1group24/mocks"
)

// UserIndex using MockSkipList
type userIndexSkiplist struct {
	skiplist *mocks.MockSL[string, Session]
}

func (u *userIndexSkiplist) Find(user string) (Session, bool) {
	return u.skiplist.Find(user)
}

func (u *userIndexSkiplist) Upsert(user string, check index_utils.UpdateCheck[string, Session]) (bool, error) {
	updated, err := u.skiplist.Upsert(user, check)
	if err != nil {
		return false, err
	}
	newSession, _ := u.skiplist.Find(user)
	fmt.Printf("UserIndexSkiplist Upsert: user=%s session=%+v\n", user, newSession)
	return updated, nil
}

func (u *userIndexSkiplist) Remove(user string) (Session, bool) {
	return u.skiplist.Remove(user)
}

// TokenIndex using MockSkipList
type tokenIndexSkiplist struct {
	skiplist *mocks.MockSL[string, string]
}

func (t *tokenIndexSkiplist) Find(token string) (string, bool) {
	return t.skiplist.Find(token)
}

func (t *tokenIndexSkiplist) Upsert(token string, check index_utils.UpdateCheck[string, string]) (bool, error) {
	updated, err := t.skiplist.Upsert(token, check)
	if err != nil {
		return false, err
	}
	newUser, _ := t.skiplist.Find(token)
	fmt.Printf("TokenIndexSkiplist Upsert: token=%s user=%s\n", token, newUser)
	return updated, nil
}

func (t *tokenIndexSkiplist) Remove(token string) (string, bool) {
	return t.skiplist.Remove(token)
}

// Setup function for creating an AuthStruct with skiplist-based indexes
func setupAuth() *AuthStruct {
	userToToken := &userIndexSkiplist{skiplist: mocks.NewMockSL[string, Session]()}

	return New(userToToken, "")
}

// Test CreateSession with MockSkipList
func TestCreateSession_WithMockSkipList(t *testing.T) {
	service := setupAuth()

	// Test creating a session for a user
	token, err := service.CreateSession("testuser")
	if err != nil {
		t.Errorf("Failed to create session: %s", err)
	}

	// We can't access userToToken directly, but we can check if the session was created by validating it
	_, err = service.ValidateSession(token)
	if err != nil {
		t.Errorf("Failed to validate session: %s", err)
	}
}

// Test ValidateSession with MockSkipList
func TestValidateSession_WithMockSkipList(t *testing.T) {
	service := setupAuth()

	// Create a session to test validation
	token, _ := service.CreateSession("testuser")

	// Test validating a valid session
	user, err := service.ValidateSession(token)
	if err != nil {
		t.Errorf("Failed to validate session: %s", err)
	}
	if user != "testuser" {
		t.Errorf("Expected user: %s, got: %s", "testuser", user)
	}

	// Test validating an invalid session
	_, err = service.ValidateSession("invalidtoken")
	if err == nil {
		t.Errorf("Expected error for invalid token")
	}
}

// Test Logout with MockSkipList
func TestLogout(t *testing.T) {
	service := setupAuth()

	// Create a session to test logout
	token, _ := service.CreateSession("testuser")

	// Test logging out the session
	success, err := service.Logout(token)
	if err != nil {
		t.Errorf("Failed to log out session: %s", err)
	}
	if !success {
		t.Errorf("Expected session to be logged out")
	}

	// Test logging out an invalid session
	success, err = service.Logout("invalidtoken")
	if err == nil {
		t.Errorf("Expected error for invalid token")
	}
	if success {
		t.Errorf("Expected logout to fail for invalid token")
	}
}

func TestNew(t *testing.T) {
	tokenToUser := &userIndexSkiplist{skiplist: mocks.NewMockSL[string, Session]()}

	New(tokenToUser, "testtokens.json")
}
func TestNewBadTokens(t *testing.T) {
	tokenToUser := &userIndexSkiplist{skiplist: mocks.NewMockSL[string, Session]()}

	New(tokenToUser, "badTokens.json")

}

func TestAuthStruct_Login(t *testing.T) {
	tokenToUser := &userIndexSkiplist{skiplist: mocks.NewMockSL[string, Session]()}

	a := New(tokenToUser, "testtokens.json")

	_, err := a.Login("Fernando")
	if err != nil {
		t.Errorf("Login failed")
	}

}

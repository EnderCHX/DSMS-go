package auth

import (
	"testing"
)

func TestLogin(t *testing.T) {
	refresh_token, access_token, err := Login("test", "test1")
	if err != nil {
		t.Error(err)
	}
	t.Log(refresh_token, access_token)
}

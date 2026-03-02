package action

import (
	"context"
	"testing"
)

type Login struct {
	Username string
	Password string
}

type LoginResult struct {
	Token string
}

func login(ctx context.Context, params *Login) (*Output[LoginResult], error) {
	return nil, nil
}

func TestAction(t *testing.T) {
	group := NewGroup("rrsc/internal/action.", nil)
	JSONApi(group, login, nil)
}

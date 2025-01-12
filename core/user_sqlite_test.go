package core

import "testing"

type UserFixture struct {
	*BaseFixture
	userStore UserStore
}

func NewUserFixture(t *testing.T) *UserFixture {
	base := NewBaseFixture(t)
	userStore := NewSqlieUserStore(base.db)
	return &UserFixture{
		BaseFixture: base,
		userStore:   userStore,
	}
}

func TestCreateUser(t *testing.T) {
}

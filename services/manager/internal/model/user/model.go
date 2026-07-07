package user

import (
	"errors"

	domaintypes "github.com/nilafzar/agents/services/manager/internal/model/internal/types"
)

type UserRole int8

const (
	_ UserRole = iota
	AdminRole
	DefaultRole
)

type User struct {
	ID           int64    `json:"id"`
	Username     string   `json:"username"`
	FirstName    *string  `json:"firstname"`
	LastName     *string  `json:"lastname"`
	Role         UserRole `json:"role"`
	EmailAddress *string  `json:"emailAddress"`
	VerifiedAt   *int64   `json:"verifiedAt"`
	PasswordHash string   `json:"-"`
	CreatedAt    int64    `json:"createdAt"`
	UpdatedAt    int64    `json:"updatedAt"`
}

type Query struct {
	*domaintypes.CursorPagination
	Username     *string   `query:"username"`
	FirstName    *string   `query:"firstname"`
	LastName     *string   `query:"lastname"`
	EmailAddress *string   `query:"emailAddress"`
	Role         *UserRole `query:"role"`
	IsVerified   *bool     `query:"isVerified"`
}

type Repository = domaintypes.Repository[User, int64, Query]

// UserRole to string
var urts = map[UserRole][]byte{
	AdminRole:   []byte("\"admin\""),
	DefaultRole: []byte("\"default\""),
}

func (ur UserRole) MarshalJSON() ([]byte, error) {
	b, ok := urts[ur]
	if !ok {
		return nil, errors.New("invalid user role")
	}
	return b, nil
}

// UserRole from string
var urfs = map[string]UserRole{
	"admin":   AdminRole,
	"default": DefaultRole,
}

func (ur *UserRole) UnmarshalJSON(b []byte) error {
	str := string(b)
	strLen := len(str)

	if strLen < 3 {
		return errors.New("user role value is too short")
	}

	str = str[1 : strLen-1]
	val, ok := urfs[str]
	if !ok {
		return errors.New("invalid user role")
	}

	*ur = val
	return nil
}

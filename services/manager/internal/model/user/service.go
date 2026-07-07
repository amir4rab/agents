package user

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/nilafzar/agents/services/manager/internal/model/internal/password"
	domaintypes "github.com/nilafzar/agents/services/manager/internal/model/internal/types"
)

type Service struct {
	db   *sql.DB
	user Repository
}

type CreateUserPayload struct {
	Username     string   `json:"-"`
	FirstName    string   `json:"-"`
	LastName     string   `json:"-"`
	EmailAddress string   `json:"-"`
	Password     string   `json:"-"`
	Role         UserRole `json:"-"`
}

type UpdateUserPayload struct {
	ID           int64  `json:"-"`
	Username     string `json:"-"`
	FirstName    string `json:"-"`
	LastName     string `json:"-"`
	EmailAddress string `json:"-"`
}

var (
	readonlyTx = &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: true}
	defaultTx  = &sql.TxOptions{Isolation: sql.LevelDefault}
)

// Create stores the user recored based on the provided data in the database
func (s *Service) Create(ctx context.Context, payload *CreateUserPayload) (*User, error) {
	u := User{}

	phc, err := password.CreateHash(payload.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()

	// Required values
	u.Username = payload.Username
	u.Role = payload.Role
	u.PasswordHash = *phc
	u.CreatedAt = now
	u.UpdatedAt = now

	// Optional values
	if payload.FirstName != "" {
		u.FirstName = &payload.FirstName
	}
	if payload.LastName != "" {
		u.LastName = &payload.LastName
	}
	if payload.EmailAddress != "" {
		u.EmailAddress = &payload.EmailAddress
	}

	tx, err := s.db.BeginTx(ctx, defaultTx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	if err = s.user.Create(tx, &u); err != nil {
		return nil, err
	}

	// Validating that the id is populated by the repository
	if u.ID == 0 {
		return nil, errors.New("id field wasn't populated by the repository")
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &u, nil
}

// Get returns a user based on the provided userID
func (s *Service) Get(ctx context.Context, userID int64) (*User, error) {
	tx, err := s.db.BeginTx(ctx, readonlyTx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	user, err := s.user.Get(tx, userID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return user, nil
}

// Find returns a page of users based on the provided queries
func (s *Service) Find(ctx context.Context, query *Query) (*domaintypes.Page[User], error) {
	tx, err := s.db.BeginTx(ctx, readonlyTx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	page, err := s.user.Find(tx, query)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return page, nil
}

// Update edits the user info based on the provided payload
func (s *Service) Update(ctx context.Context, payload *UpdateUserPayload) (*User, error) {
	tx, err := s.db.BeginTx(ctx, defaultTx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	u, err := s.user.Get(tx, payload.ID)
	if err != nil {
		return nil, err
	}

	u.Username = payload.Username

	if payload.FirstName != "" {
		u.FirstName = &payload.FirstName
	} else {
		u.FirstName = nil
	}

	if payload.LastName != "" {
		u.LastName = &payload.LastName
	} else {
		u.LastName = nil
	}

	if payload.EmailAddress != "" {
		u.EmailAddress = &payload.EmailAddress
	} else {
		u.EmailAddress = nil
	}

	now := time.Now().UnixMilli()
	u.UpdatedAt = now

	if err = s.user.Update(tx, u); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return u, nil
}

// Delete removes the user form the database
func (s *Service) Delete(ctx context.Context, userID int64) (bool, error) {
	tx, err := s.db.BeginTx(ctx, defaultTx)
	if err != nil {
		return false, err
	}

	defer tx.Rollback()

	err = s.user.Delete(tx, userID)
	if err != nil {
		return false, err
	}

	if err = tx.Commit(); err != nil {
		return false, err
	}

	return true, nil
}

// NewService creates a new user service
func NewService(sql *sql.DB, repo Repository) *Service {
	return &Service{sql, repo}
}

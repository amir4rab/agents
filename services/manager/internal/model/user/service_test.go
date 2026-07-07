package user_test

import (
	"database/sql"
	"fmt"
	"math"
	"testing"

	"github.com/nilafzar/agents/services/manager/internal/model/internal/sqlite"
	domaintypes "github.com/nilafzar/agents/services/manager/internal/model/internal/types"
	"github.com/nilafzar/agents/services/manager/internal/model/user"
)

func newTestService(t *testing.T) (*user.Service, *sql.DB) {
	t.Helper()

	db, err := sqlite.Conn(sqlite.NewMemConfig())
	if err != nil {
		t.Fatal(err)
	}

	repo := user.NewRepository()
	svc := user.NewService(db, repo)

	return svc, db
}

func createTestUser(t *testing.T, svc *user.Service, username string, role user.UserRole) *user.User {
	t.Helper()

	u, err := svc.Create(t.Context(), &user.CreateUserPayload{
		Username:  username,
		FirstName: "Test",
		LastName:  "User",
		Role:      role,
		Password:  "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	return u
}

func TestCreate(t *testing.T) {
	svc, _ := newTestService(t)

	u, err := svc.Create(t.Context(), &user.CreateUserPayload{
		Username:  "johndoe",
		FirstName: "John",
		LastName:  "Doe",
		Role:      user.AdminRole,
		Password:  "securepassword123",
	})
	if err != nil {
		t.Fatal(err)
	}

	if u.ID == 0 {
		t.Error("Expected a non-zero ID to be assigned")
	}

	if u.Username != "johndoe" {
		t.Error("Expected username to be set")
	}

	if u.Role != user.AdminRole {
		t.Error("Expected role to be AdminRole")
	}

	if u.PasswordHash == "" {
		t.Error("Expected password hash to be set")
	}

	if u.FirstName == nil || *u.FirstName != "John" {
		t.Error("Expected first name to be set")
	}

	if u.LastName == nil || *u.LastName != "Doe" {
		t.Error("Expected last name to be set")
	}

	if u.CreatedAt == 0 {
		t.Error("Expected created at to be set")
	}

	if u.UpdatedAt == 0 {
		t.Error("Expected updated at to be set")
	}
}

func TestCreateWithOnlyRequiredFields(t *testing.T) {
	svc, _ := newTestService(t)

	u, err := svc.Create(t.Context(), &user.CreateUserPayload{
		Username: "janedoe",
		Role:     user.DefaultRole,
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	if u.ID == 0 {
		t.Error("Expected a non-zero ID to be assigned")
	}

	if u.FirstName != nil {
		t.Error("Expected first name to be nil when not provided")
	}

	if u.LastName != nil {
		t.Error("Expected last name to be nil when not provided")
	}

	if u.EmailAddress != nil {
		t.Error("Expected email address to be nil when not provided")
	}
}

func TestCreateWithAllOptionalFields(t *testing.T) {
	svc, _ := newTestService(t)

	email := "john@example.com"
	u, err := svc.Create(t.Context(), &user.CreateUserPayload{
		Username:     "johnny",
		FirstName:    "John",
		LastName:     "Doe",
		EmailAddress: email,
		Role:         user.DefaultRole,
		Password:     "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	if u.FirstName == nil || *u.FirstName != "John" {
		t.Error("Expected first name to be set")
	}

	if u.LastName == nil || *u.LastName != "Doe" {
		t.Error("Expected last name to be set")
	}

	if u.EmailAddress == nil || *u.EmailAddress != "john@example.com" {
		t.Error("Expected email address to be set")
	}
}

func TestCreateDuplicateUsername(t *testing.T) {
	svc, _ := newTestService(t)

	createTestUser(t, svc, "duplicate", user.DefaultRole)

	_, err := svc.Create(t.Context(), &user.CreateUserPayload{
		Username: "duplicate",
		Role:     user.DefaultRole,
		Password: "password123",
	})
	if err == nil {
		t.Error("Expected an error when creating a user with a duplicate username")
	}
}

func TestGet(t *testing.T) {
	svc, _ := newTestService(t)

	created := createTestUser(t, svc, "gettable", user.DefaultRole)

	u, err := svc.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if u.ID != created.ID {
		t.Error("Expected the same user ID")
	}

	if u.Username != "gettable" {
		t.Error("Expected the username to match")
	}
}

func TestGetNonExistent(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Get(t.Context(), 99999)
	if err == nil {
		t.Error("Expected an error when getting a non-existent user")
	}
}

func TestGetZeroID(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Get(t.Context(), 0)
	if err == nil {
		t.Error("Expected an error when getting a user with ID 0")
	}
}

func TestUpdate(t *testing.T) {
	svc, _ := newTestService(t)

	created := createTestUser(t, svc, "oldusername", user.DefaultRole)

	u, err := svc.Update(t.Context(), &user.UpdateUserPayload{
		ID:        created.ID,
		Username:  "newusername",
		FirstName: "Updated",
		LastName:  "Name",
	})
	if err != nil {
		t.Fatal(err)
	}

	if u.Username != "newusername" {
		t.Error("Expected username to be updated")
	}

	if u.FirstName == nil || *u.FirstName != "Updated" {
		t.Error("Expected first name to be updated")
	}

	if u.LastName == nil || *u.LastName != "Name" {
		t.Error("Expected last name to be updated")
	}

	if u.UpdatedAt == created.UpdatedAt {
		t.Error("Expected updated at to change")
	}
}

func TestUpdateClearOptionalFields(t *testing.T) {
	svc, _ := newTestService(t)

	email := "remove@example.com"
	created, err := svc.Create(t.Context(), &user.CreateUserPayload{
		Username:     "clearfields",
		FirstName:    "HasFirstName",
		LastName:     "HasLastName",
		EmailAddress: email,
		Role:         user.DefaultRole,
		Password:     "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	u, err := svc.Update(t.Context(), &user.UpdateUserPayload{
		ID:           created.ID,
		Username:     "clearfields",
		FirstName:    "",
		LastName:     "",
		EmailAddress: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	if u.FirstName != nil {
		t.Error("Expected first name to be cleared")
	}

	if u.LastName != nil {
		t.Error("Expected last name to be cleared")
	}

	if u.EmailAddress != nil {
		t.Error("Expected email address to be cleared")
	}
}

func TestUpdateNonExistent(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Update(t.Context(), &user.UpdateUserPayload{
		ID:       99999,
		Username: "nobody",
	})
	if err == nil {
		t.Error("Expected an error when updating a non-existent user")
	}
}

func TestDelete(t *testing.T) {
	svc, _ := newTestService(t)

	created := createTestUser(t, svc, "deleteme", user.DefaultRole)

	ok, err := svc.Delete(t.Context(), created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Error("Expected Delete to return true")
	}

	_, err = svc.Get(t.Context(), created.ID)
	if err == nil {
		t.Error("Expected an error when getting a deleted user")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Delete(t.Context(), 99999)
	if err == nil {
		t.Error("Expected an error when deleting a non-existent user")
	}
}

func TestDeleteZeroID(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Delete(t.Context(), 0)
	if err == nil {
		t.Error("Expected an error when deleting a user with ID 0")
	}
}

func TestFindEmptyResult(t *testing.T) {
	svc, _ := newTestService(t)

	page, err := svc.Find(t.Context(), &user.Query{
		CursorPagination: &domaintypes.CursorPagination{First: 25},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 0 {
		t.Error("Expected no items in the result")
	}

	if page.Info.HasNextPage {
		t.Error("Expected no next page for empty results")
	}

	if page.Info.HasPreviousPage {
		t.Error("Expected no previous page for empty results")
	}
}

func TestFindByUsername(t *testing.T) {
	svc, _ := newTestService(t)

	createTestUser(t, svc, "alice", user.DefaultRole)
	createTestUser(t, svc, "bob", user.DefaultRole)
	createTestUser(t, svc, "charlie", user.DefaultRole)

	username := "bob"
	page, err := svc.Find(t.Context(), &user.Query{Username: &username})
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(page.Items))
	}

	if page.Items[0].Username != "bob" {
		t.Error("Expected the username to match 'bob'")
	}
}

func TestFindByUsernameNoMatch(t *testing.T) {
	svc, _ := newTestService(t)

	createTestUser(t, svc, "alice", user.DefaultRole)

	username := "nonexistent"
	page, err := svc.Find(t.Context(), &user.Query{Username: &username})
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 0 {
		t.Error("Expected no users to match")
	}
}

func TestFindByRole(t *testing.T) {
	svc, _ := newTestService(t)

	createTestUser(t, svc, "admin1", user.AdminRole)
	createTestUser(t, svc, "admin2", user.AdminRole)
	createTestUser(t, svc, "user1", user.DefaultRole)

	role := user.AdminRole
	page, err := svc.Find(t.Context(), &user.Query{Role: &role})
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 2 {
		t.Fatalf("Expected 2 admin users, got %d", len(page.Items))
	}
}

func TestFindPagination(t *testing.T) {
	svc, _ := newTestService(t)

	users := make([]*user.User, 250)
	for i := range 250 {
		username := fmt.Sprintf("user-%d", i)
		u := createTestUser(t, svc, username, user.DefaultRole)
		users[i] = u
	}

	page, err := svc.Find(t.Context(), &user.Query{CursorPagination: &domaintypes.CursorPagination{First: 25}})
	if err != nil {
		t.Fatal(err)
	}

	defaultLimit := page.Info.Limit
	if defaultLimit == 0 {
		defaultLimit = 25
	}

	totalPages := int(math.Ceil(float64(len(users)) / float64(defaultLimit)))
	nextCursor := page.Info.EndCursor
	previousCursor := page.Info.StartCursor

	t.Run("forward", func(t *testing.T) {
		for p := range totalPages - 1 {
			page, err := svc.Find(t.Context(), &user.Query{
				CursorPagination: &domaintypes.CursorPagination{After: nextCursor},
			})
			if err != nil {
				t.Fatal(err)
			}

			if p+1 != totalPages-1 {
				nextCursor = page.Info.EndCursor
				continue
			}

			if page.Info.HasNextPage {
				t.Error("Expected the last page not to have a next page")
			}
		}
	})

	t.Run("backward", func(t *testing.T) {
		for page := totalPages; page > 0; page-- {
			result, err := svc.Find(t.Context(), &user.Query{
				CursorPagination: &domaintypes.CursorPagination{Before: previousCursor},
			})
			if err != nil {
				t.Fatal(err)
			}

			if page != 1 {
				previousCursor = result.Info.StartCursor
				continue
			}

			if result.Info.HasPreviousPage {
				t.Error("Expected the first page not to have a previous page")
			}
		}
	})

	t.Run("single page when limit exceeds total", func(t *testing.T) {
		page, err := svc.Find(t.Context(), &user.Query{
			CursorPagination: &domaintypes.CursorPagination{First: 255},
		})
		if err != nil {
			t.Fatal(err)
		}

		if page.Info.HasNextPage {
			t.Error("Expected no next page when limit exceeds total")
		}

		if len(page.Items) != len(users) {
			t.Errorf("Expected %d items, got %d", len(users), len(page.Items))
		}
	})

	t.Run("has next page when more results exist", func(t *testing.T) {
		page, err := svc.Find(t.Context(), &user.Query{
			CursorPagination: &domaintypes.CursorPagination{First: 25},
		})
		if err != nil {
			t.Fatal(err)
		}

		if !page.Info.HasNextPage {
			t.Error("Expected hasNextPage to be true when there are more results")
		}
	})
}

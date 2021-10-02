package users

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jakubDoka/keeper/core"
	"github.com/jakubDoka/keeper/state"
	"github.com/jakubDoka/keeper/util/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const (
	MaxNameLength     = 128
	MaxEmailLength    = 320
	MinPasswordLength = 8
)

var (
	ErrDuplicateEmail   = errors.New("Email is already taken.")
	ErrOperationFailed  = errors.New("Operation failed.")
	ErrAccountNotFound  = errors.New("Account not found.")
	ErrNameTooLong      = fmt.Errorf("Name is too long. Max length is %d characters.", MaxNameLength)
	ErrEmailTooLong     = fmt.Errorf("Email is too long. Max length is %d characters.", MaxEmailLength)
	ErrPasswordTooShort = fmt.Errorf("Password is too short. Min length is %d characters.", MinPasswordLength)
)

//go:embed users.sql
var sqlString string

func Initialize(a *core.App) {
	a.Prepare("users", sqlString)
}

func Create(s *state.State, user Unverified) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.Get("users:create").Exec(id.String(), user.Email, user.Password)
	if err, ok := err.(*pq.Error); ok {
		if err.Code.Name() == "unique_violation" && err.Column == "email" {
			return uuid.Nil, ErrDuplicateEmail
		}
		s.Error(err.Error())
	}
	return id, nil
}

func SetName(s *state.State, id uuid.UUID, name string) error {
	status, err := s.Get("users:set-name").Exec(name, id.String())
	if err != nil {
		s.Error(err.Error())
		return ErrOperationFailed
	}
	count, err := status.RowsAffected()
	if err != nil {
		s.Error(err.Error())
		return ErrOperationFailed
	}
	if count != 1 {
		return ErrAccountNotFound
	}

	return nil
}

func Verify(s *state.State, email, password string) (uuid.UUID, bool) {
	var storedPassword, storedID string
	err := s.Get("users:get-password-and-id").QueryRow(email).Scan(&storedPassword, &storedID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.Error(err.Error())
		}
		return uuid.Nil, false
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		return uuid.Nil, false
	}

	return uuid.MustParseWithHyphens(storedID), true
}

type Unverified struct {
	Email      string
	Password   string
	Code       uuid.UUID
	Expiration time.Time
}

func NewUnverified(email, password string) (Unverified, error) {
	if len(password) < MinPasswordLength {
		return Unverified{}, ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	if len(hash) != 60 {
		panic("Hash is not 60 bytes long.")
	}

	if len(email) > MaxEmailLength {
		return Unverified{}, ErrEmailTooLong
	}

	return Unverified{
		Email:      email,
		Password:   string(hash),
		Code:       uuid.New(),
		Expiration: time.Now().Add(time.Minute * 5),
	}, nil
}

package category

import (
	"errors"
	"time"
)

var (
	ErrNotFound            = errors.New("category not found")
	ErrInvalidCategoryType = errors.New("invalid category type")
)

type Category struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateCategoryInput struct {
	Name string
	Type string
}

type UpdateCategoryInput struct {
	Name string
	Type string
}

func IsValidType(categoryType string) bool {
	switch categoryType {
	case "income", "expense":
		return true
	default:
		return false
	}
}

func NotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

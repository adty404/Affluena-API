package category

import (
	"errors"
	"time"
)

var (
	ErrNotFound            = errors.New("category not found")
	ErrInvalidCategoryType = errors.New("invalid category type")
	ErrParentTypeMismatch  = errors.New("invalid: parent category must have the same type")
	// "repeated", not "duplicate": httpx.WriteError keyword-maps "duplicate"
	// to 409 Conflict before the "invalid" → 400 match (see httpx/response.go).
	ErrDuplicateReorderID = errors.New("invalid reorder request: repeated category id")
)

type Category struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ParentID  *string   `json:"parent_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Icon      string    `json:"icon"`
	Color     string    `json:"color"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateCategoryInput struct {
	Name     string
	Type     string
	ParentID *string
	Icon     string
	Color    string
}

type UpdateCategoryInput struct {
	Name     string
	Type     string
	ParentID *string
	Icon     string
	Color    string
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

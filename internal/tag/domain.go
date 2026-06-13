package tag

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("tag not found")

type Tag struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateTagInput struct {
	Name string `json:"name" binding:"required"`
}

type UpdateTagInput struct {
	Name string `json:"name" binding:"required"`
}

func NotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

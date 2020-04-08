package model

import (
	"github.com/google/uuid"
	"time"
)

// Model is default model
type Model struct {
	ID        uuid.UUID `db:"id,primarykey"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

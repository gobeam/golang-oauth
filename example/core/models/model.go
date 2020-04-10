package models

import (
	"github.com/jinzhu/gorm"
	"time"
)

type Model struct {
	ID        uint      `json:"id" gorm:"primary_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var DB *gorm.DB

func InitializeDb(db *gorm.DB) {
	DB = db
	DB.AutoMigrate(&User{}, &Category{}, &Post{})
}

package models

import (
	"gorm.io/gorm"
)

// CreateAdminWithAuthor creates a user with admin username and password, then creates an author with the same name and links them.
func CreateAdminWithAuthor(db *gorm.DB, username, password string) (*User, *Author, error) {
	user := &User{Username: username, Password: password}
	if err := db.Create(user).Error; err != nil {
		return nil, nil, err
	}
	author := &Author{Name: username, UserID: user.ID}
	if err := db.Create(author).Error; err != nil {
		return user, nil, err
	}
	return user, author, nil
}

// GetOrCreateTag returns the tag with the given name, or creates it if it doesn't exist.
func GetOrCreateTag(db *gorm.DB, name, slug string) (*Tag, error) {
	var tag Tag
	if err := db.Where("name = ?", name).First(&tag).Error; err == nil {
		return &tag, nil
	}
	tag = Tag{Name: name, Slug: slug}
	if err := db.Create(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

type User struct {
	gorm.Model
	Username string  `gorm:"unique;not null" json:"username"`
	Password string  `gorm:"not null" json:"password"`
	Author   *Author // Optional: reverse relation
}

func (u *User) CreateUser(db *gorm.DB, username, password string) error {
	u.Username = username
	u.Password = password
	return db.Create(u).Error
}

func (u *User) CreateAuthor(db *gorm.DB, name string) error {
	author := Author{Name: name, UserID: u.ID}
	return db.Create(&author).Error
}

func EnsureGoTagExists(db *gorm.DB) error {
	var tag Tag
	if err := db.Where("name = ?", "Go").First(&tag).Error; err != nil {
		return db.Create(&Tag{Name: "Go", Slug: "go"}).Error
	}
	return nil
}

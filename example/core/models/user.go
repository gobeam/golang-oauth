package models

type User struct {
	Model
	Name     string `json:"name" binding:"required,max=100,min=2"`
	Email    string `json:"email" binding:"required,email,uniqueEmail" gorm:"type:varchar(200);unique_index"`
	Password string `json:"password" binding:"required,min=6,max=20"`
}

func (u *User) FindById() {
	DB.First(&u, u.ID)
}

func (u *User) FindByEmail() {
	DB.Where(&u).First(&u)
}

func (u *User) Create() {
	DB.Create(&u)
}

package models

type Banner struct {
	Model
	Name      string `json:"name" binding:"required,max=100,min=2"`
	Title      string `json:"title" binding:"required,max=100,min=2"`
	AsciiImage string `sql:"type:MEDIUMTEXT;" json:"ascii_image" binding:"required,min=2"`
	Position   int64  `json:"position"`
	Status     string `json:"status"`
	Label      string `json:"label"`
}

func (u *Banner) FindById() {
	DB.First(&u, ID)
}


func (u *Banner) Create() {
	DB.Create(&u)
}

func (u *Banner) Update() {
	DB.Save(&u)
}

package models

type Category struct {
	Model
	Name   string `json:"name" binding:"required,max=100,min=2"`
	Status string `json:"status" binding:"required"`
	Label  string `json:"label" binding:"required"`
}

type Categories []Category

func (c *Category) FindById(id uint) {
	DB.First(&c, id)
}

func (c *Categories) Get() {
	DB.Find(&c)
}

func (c *Category) Create() {
	DB.Create(&c)
}

func (c *Category) Update() {
	DB.Save(&c)
}

func (c *Category) Delete() {
	DB.Delete(&c)
}

package models

type Post struct {
	Model
	Title       string      `json:"title" binding:"required,max=100,min=2"`
	Description string      `sql:"type:text;" json:"description" binding:"required,max=100,min=2"`
	OgType      string      `json:"og_type" binding:"required,max=100,min=2"`
	OgUrl       string      `json:"og_url" binding:"required,max=100,min=2"`
	Image       string      `json:"image"`
	Body        string      `sql:"type:text;" json:"body" binding:"required,min=2"`
	Category    []*Category `gorm:"many2many:post_categories;"`
	Profanity   bool        `json:"profanity" binding:"required"`
	UserId      string      `json:"user_id" binding:"required"`
	Status      string      `json:"status" binding:"required"`
}

type Posts []Post

func (p *Post) FindById(id int64) {
	DB.First(&p, id)
}

func (p *Post) Create() {
	DB.Create(&p)
}

func (p *Posts) Get() {
	DB.Find(&p)
}

func (p *Post) Update() {
	DB.Save(&p)
}

func (p *Post) Delete() {
	DB.Delete(&p)
}

func (p *Post) AssignCategory(catIds []uint, update bool) {

	var categories []*Category
	for i := 0; i < len(catIds); i++ {
		category := Category{}
		category.FindById(catIds[i])
		if category.ID > 0 {
			categories = append(categories, &category)
		}
	}

	if len(categories) > 0 {
		if update {
			DB.Model(&p).Association("Category").Replace(&categories)
		} else {
			DB.Model(&p).Association("Category").Append(&categories)
		}
	}
}

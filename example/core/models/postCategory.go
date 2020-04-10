package models

type PostCategory struct {
	Model
	PostId     uint `json:"post_id" binding:"required"`
	CategoryId uint `json:"category_id" binding:"required"`
}

type PostCategories []PostCategory

func (p *PostCategory) Create() {
	DB.Create(&p)
}

func (p *PostCategories) GetByPostId(id int64) {
	DB.Where("post_id = ?", id).Find(&p)
}

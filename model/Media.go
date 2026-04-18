package model

type Media struct {
	MediaID   uint    `gorm:"primary_key;auto_increment;not null;unique;column:media_id" json:"media_id"`
	MimeType  string  `gorm:"not null;type:varchar(255);column:mime_type" json:"mime_type"`
	Url       string  `gorm:"not null;type:varchar(255);column:url" json:"url"`
	Alt       string  `gorm:"not null;type:varchar(255);column:alt" json:"alt"`
	Slug      string  `gorm:"not null;type:varchar(255);column:slug" json:"slug"`
	CheckSum  string  `gorm:"not null;type:varchar(255);column:check_sum" json:"check_sum"`
	IsActive  bool    `gorm:"not null;type:bool;column:is_active" json:"is_active"`
	CreatedAt string  `gorm:"not null;type:date;column:created_at" json:"created_at"`
	CreatedBy uint    `gorm:"not null;type:int;column:created_by" json:"created_by"`
	UpdatedAt *string `gorm:"null;type:date;column:updated_at" json:"updated_at"`
	UpdatedBy *uint   `gorm:"null;type:int;column:updated_by" json:"updated_by"`
}

func (Media) TableName() string {
	return "media"
}

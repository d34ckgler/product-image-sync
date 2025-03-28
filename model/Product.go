package model

import "time"

type Product struct {
	ProductID   uint       `gorm:"primary_key; auto_increment; not null; unique; column:product_id" json:"product_id"`
	Sku         string     `gorm:"not null; type:varchar(6); unique; column:sku" json:"sku"`
	Name        string     `gorm:"not null; type:varchar(255); column:name" json:"name"`
	Description string     `gorm:"not null; type:varchar(255); column:description" json:"description"`
	Tax         float64    `gorm:"not null; type:float; column:tax" json:"tax"`
	Stock       int        `gorm:"not null; type: int; column:stock" json:"stock"`
	Price       float64    `gorm:"not null; type:float; column:price" json:"price"`
	ImageID     *uint      `gorm:"null; foreignkey:MediaID; type:int; column:image_id" json:"image_id,omitempty"`
	IsActive    bool       `gorm:"not null; type:bool; column:is_active" json:"is_active"`
	CreatedAt   time.Time  `gorm:"not null; type:date; column:created_at" json:"created_at"`
	CreatedBy   uint       `gorm:"not null; type:int; column:created_by" json:"created_by"`
	UpdatedAt   *time.Time `gorm:"null; type:date; column:updated_at" json:"updated_at"`
	UpdatedBy   *uint      `gorm:"null; type:int; column:updated_by" json:"updated_by"`
}

func (Product) TableName() string {
	return "product"
}

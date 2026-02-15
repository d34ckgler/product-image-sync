package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/d34ckgler/sync-product-image/model"
	"github.com/supabase-community/supabase-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Supabase struct {
	Host     string
	User     string
	Password string
	Port     string
	Dbname   string
	DB       *gorm.DB
	sql      *sql.DB
	Client   *supabase.Client
}

type Ecommerce interface {
	Open(client *supabase.Client)
	InitPool() (*gorm.DB, error)
	Connect()
	GetProducts() ([]model.Product, error)
	GetProduct(id int) (model.Product, error)
	GetProductBySku(sku string) (model.Product, error)
	Create(value interface{}) (tx *gorm.DB)
	Save(value interface{}) (tx *gorm.DB)
	Close() error
}

func (ecm *Supabase) Open(client *supabase.Client) {
	ecm.Client = client
}

func (ecm *Supabase) InitPool() (*gorm.DB, error) {
	// dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/Caracas application_name=ecommerce", ecm.Host, ecm.User, ecm.Password, ecm.Dbname, ecm.Port)
	// dsn := "postgresql://postgres.gpacszhbwbrdjyyxlzdr:6Dv0YvCiGBJmNYT3@aws-0-us-west-1.pooler.supabase.com:5432/postgres"

	dsn := "postgresql://postgres.bmjqebmyyeybsyzwirkd:6Dv0YvCiGBJmNYT3@aws-0-us-east-1.pooler.supabase.com:5432/postgres"
	// dsn := "postgresql://postgres:postgres@127.0.0.1:54322/postgres"

	// sqlDB, err := sql.Open("pgx", dsn)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create connection pool: %v", err)
	// }

	// sqlDB.SetMaxIdleConns(10)
	// sqlDB.SetMaxOpenConns(100)
	// sqlDB.SetConnMaxLifetime(30)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	ecm.sql, err = db.DB()
	if err != nil {
		return nil, err
	}
	ecm.DB = db

	return db, nil
}

func (ecm *Supabase) Connect() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/Caracas application_name=ecommerce", ecm.Host, ecm.User, ecm.Password, ecm.Dbname, ecm.Port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: false,
		Logger:      nil,
	})
	if err != nil {
		panic("failed to connect database server")
	}

	ecm.sql, err = db.DB()
	if err != nil {
		panic(err)
	}

	ecm.DB = db
}

func (ecm *Supabase) GetProducts() (products []model.Product, err error) {
	fmt.Println("Attempting to load paginated products from Supabase...")
	offset := 0

	for {
		var tempProducs []model.Product
		data, _, err := ecm.Client.From("product").
			Select("*", "", false).
			// Is("image_id", "null").
			Range(offset, offset+1000-1, "").
			Execute()
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(data, &tempProducs)
		if err != nil {
			return nil, fmt.Errorf("error al deserializar los datos JSON a productos: %w", err)
		}

		products = append(products, tempProducs...)
		fmt.Println("Loaded", len(tempProducs), "products from Supabase.")

		if len(tempProducs) == 0 {
			break
		}

		offset += 1000
	}

	fmt.Printf("Loaded %d products from Supabase.\n", len(products))

	return products, nil
}

func (ecm *Supabase) Desallocate() {
	_, err := ecm.sql.Exec("DEALLOCATE ALL;")
	if err != nil {
		panic(err)
	}
}

func (ecm *Supabase) GetProduct(id int) (product model.Product) {
	ecm.DB.First(&product, id)

	return
}

func (ecm *Supabase) GetMedia(slug string) (media model.Media) {
	// ecm.DB.First(&media, "slug = ?", slug)
	ecm.Client.From("media").Select("*", "", false).Eq("slug", slug).Single().ExecuteTo(&media)

	return
}

func (ecm *Supabase) GetProductBySku(sku string) (product model.Product) {
	ecm.DB.Where("sku = ?", sku).Find(&product)

	return
}

func (ecm *Supabase) Create(value interface{}) (tx *gorm.DB) {
	// ecm.Desallocate()
	return ecm.DB.Create(value)
}

func (ecm *Supabase) Save(value interface{}) (tx *gorm.DB) {
	// ecm.Desallocate()
	return ecm.DB.Save(value)
}

func (ecm *Supabase) Close() error {
	sql, err := ecm.DB.DB()
	if err != nil {
		return err
	}

	return sql.Close()
}

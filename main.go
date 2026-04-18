package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"time"

	"github.com/chai2010/webp"
	"github.com/d34ckgler/sync-product-image/database"
	"github.com/d34ckgler/sync-product-image/model"
	"github.com/joho/godotenv"
	storage_go "github.com/supabase-community/storage-go"
	"github.com/supabase-community/supabase-go"
)

var sb *database.Supabase
var client *supabase.Client
var SUPABASE_URL, SUPABASE_KEY, SUPABASE_PUBLIC_MEDIA_URL, API_BASE_URL, ENDPOINT_MEDIA, WATCH_FILE_PATH, OUTPUT_FILE_PATH string
var err error

func init() {
	err = godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading environment variables")
	}

	// Initialize environment variables values
	SUPABASE_URL = os.Getenv("SUPABASE_URL")
	SUPABASE_KEY = os.Getenv("SUPABASE_KEY")
	API_BASE_URL = os.Getenv("API_BASE_URL")
	ENDPOINT_MEDIA = os.Getenv("ENDPOINT_MEDIA")
	WATCH_FILE_PATH = os.Getenv("WATCH_FILE_PATH")
	OUTPUT_FILE_PATH = os.Getenv("OUTPUT_FILE_PATH")
	SUPABASE_PUBLIC_MEDIA_URL = os.Getenv("SUPABASE_PUBLIC_MEDIA_URL")
	// TOKEN = os.Getenv("TOKEN")

	// Initialize Supabase client
	client, err = supabase.NewClient(SUPABASE_URL, SUPABASE_KEY, nil)
	if err != nil {
		log.Fatal("Error initializing Supabase client: ", err)
	}
	fmt.Println("Supabase client initialized")
	// end initialize environment variables
}

// calculateChecksum calcula el SHA256 de los bytes proporcionados
func calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// findSourceFile busca la imagen fuente (PNG o JPG) y retorna su path y contenido
func findSourceFile(sku string) (string, bool) {
	// Intentar PNG primero
	pngPath := WATCH_FILE_PATH + "/" + sku + ".png"
	if _, err := os.Stat(pngPath); err == nil {
		return pngPath, false
	}
	// Intentar JPG
	jpgPath := WATCH_FILE_PATH + "/" + sku + ".jpg"
	if _, err := os.Stat(jpgPath); err == nil {
		return jpgPath, true
	}
	return "", false
}

func convertToWebp(sku string, forceReconvert bool) ([]byte, error) {
	// convert
	var buf bytes.Buffer
	isJPG := false

	// Si ya existe la imagen en webp y no se fuerza reconversión, usar caché
	if !forceReconvert {
		if _, err := os.Stat(OUTPUT_FILE_PATH + "/" + sku + ".webp"); err == nil {
			fmt.Printf("[%s]: image already exists in cache\n", sku)
			file, err := os.Open(OUTPUT_FILE_PATH + "/" + sku + ".webp")
			if err != nil {
				return nil, err
			}
			defer file.Close()
			_, err = io.Copy(&buf, file)
			if err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
	} else {
		// Eliminar caché WebP antiguo si existe
		os.Remove(OUTPUT_FILE_PATH + "/" + sku + ".webp")
	}

	file, err := os.Open(WATCH_FILE_PATH + "/" + sku + ".png")
	if err != nil {
		// fmt.Println("Error opening file: ", err)
		// return nil, err
		isJPG = true
		file, err = os.Open(WATCH_FILE_PATH + "/" + sku + ".jpg")
		if err != nil {
			// fmt.Println("Error opening file: ", err)
			return nil, err
		}
	}
	defer file.Close()

	// Decode the JPG image
	var img image.Image
	if isJPG {
		img, err = jpeg.Decode(file)
	} else {
		img, err = png.Decode(file)
	}
	if err != nil {
		// fmt.Println(err)
		return nil, err
	}

	// Convert the image to WebP format
	err = webp.Encode(&buf, img, &webp.Options{Quality: 80})
	if err != nil {
		// fmt.Println(err)
		return nil, err
	}

	// Save the WebP image to a file
	err = os.WriteFile(OUTPUT_FILE_PATH+"/"+sku+".webp", buf.Bytes(), 0644)
	if err != nil {
		// fmt.Println(err)
		return nil, err
	}
	return buf.Bytes(), nil
}


func uploadToBucket(filename string, data []byte, upsert bool) error {
	bucketPath := fmt.Sprintf("/products/assets/webp/%s", filename)
	mimeType := "image/webp"

	errorMessage, err := client.Storage.UploadFile("media", bucketPath, bytes.NewReader(data), storage_go.FileOptions{
		ContentType: &mimeType,
		Upsert:      &upsert,
	})
	if err != nil {
		fmt.Printf("Error uploading file to bucket: %s, %s\n", errorMessage, err)
		return err
	}
	action := "uploaded"
	if upsert {
		action = "replaced (upsert)"
	}
	fmt.Printf("File %s successfully to bucket: %s\n", action, bucketPath)
	return nil
}

func main() {
	sb = &database.Supabase{
		Host:     os.Getenv("sp_host"),
		User:     os.Getenv("sp_user"),
		Password: os.Getenv("sp_password"),
		Port:     os.Getenv("sp_port"),
		Dbname:   os.Getenv("sp_dbname"),
	}

	sb.Open(client)

	// Read All Products - supabase
	products, err := sb.GetProducts()
	if err != nil {
		log.Fatal(err)
	}

	updated := 0
	insertedCount := 0
	skipped := 0

	for _, product := range products {
		media := sb.GetMedia(product.Sku)

		// Calcular checksum de la imagen fuente (PNG/JPG)
		sourceChecksum := ""
		sourceData, sourceErr := os.ReadFile(WATCH_FILE_PATH + "/" + product.Sku + ".png")
		if sourceErr != nil {
			sourceData, sourceErr = os.ReadFile(WATCH_FILE_PATH + "/" + product.Sku + ".jpg")
		}
		if sourceErr != nil {
			// No existe imagen fuente para este SKU
			continue
		}
		sourceChecksum = calculateChecksum(sourceData)

		if media.MediaID == 0 {
			// ===== CASO 1: Media NO existe → crear nueva =====
			buf, err := convertToWebp(product.Sku, false)
			if err != nil {
				continue
			}

			err = uploadToBucket(product.Sku+".webp", buf, false)
			if err != nil {
				continue
			}

			payload := map[string]interface{}{
				"mime_type":  "webp",
				"url":        fmt.Sprintf("%s/%s.webp", SUPABASE_PUBLIC_MEDIA_URL, product.Sku),
				"alt":        product.Name,
				"slug":       product.Sku,
				"check_sum":  sourceChecksum,
				"is_active":  true,
				"created_at": time.Now().UTC().Format(time.RFC3339),
				"created_by": 1,
			}
			insertedData, _, err := sb.Client.From("media").Insert(payload, false, "", "", "").Execute()
			if err != nil {
				fmt.Printf("[%s] Error inserting media: %v\n", product.Sku, err)
				continue
			}

			newMedia := []model.Media{}
			err = json.Unmarshal(insertedData, &newMedia)
			if err != nil {
				fmt.Printf("[%s] Error unmarshalling media: %v\n", product.Sku, err)
				continue
			}

			// Vincular media al producto
			updatePayload := map[string]interface{}{
				"image_id":   newMedia[0].MediaID,
				"updated_at": time.Now().UTC().Format(time.RFC3339),
				"updated_by": 1,
			}
			_, _, err = sb.Client.From("product").Update(updatePayload, "", "").Eq("product_id", fmt.Sprintf("%d", product.ProductID)).Execute()
			if err != nil {
				fmt.Printf("[%s] Error updating product: %v\n", product.Sku, err)
			}

			fmt.Printf("[%s] ✅ New image uploaded and linked (checksum: %s)\n", product.Sku, sourceChecksum[:12])
			insertedCount++

		} else if media.CheckSum == "" {
			// ===== CASO 2: Media existe pero check_sum vacío (migración) → solo guardar checksum =====
			updatePayload := map[string]interface{}{
				"check_sum":  sourceChecksum,
				"updated_at": time.Now().UTC().Format(time.RFC3339),
				"updated_by": 1,
			}
			_, _, err := sb.Client.From("media").Update(updatePayload, "", "").Eq("media_id", fmt.Sprintf("%d", media.MediaID)).Execute()
			if err != nil {
				fmt.Printf("[%s] Error updating checksum: %v\n", product.Sku, err)
				continue
			}

			fmt.Printf("[%s] 📝 Checksum populated (migration): %s\n", product.Sku, sourceChecksum[:12])
			updated++

		} else if media.CheckSum != sourceChecksum {
			// ===== CASO 3: Media existe y checksum diferente → imagen cambió, re-subir =====
			fmt.Printf("[%s] 🔄 Checksum changed (stored: %s → new: %s)\n", product.Sku, media.CheckSum[:min(12, len(media.CheckSum))], sourceChecksum[:12])

			// Forzar reconversión (elimina caché WebP antiguo)
			buf, err := convertToWebp(product.Sku, true)
			if err != nil {
				fmt.Printf("[%s] Error converting image: %v\n", product.Sku, err)
				continue
			}

			// Subir nueva imagen con upsert (reemplaza la existente directamente)
			err = uploadToBucket(product.Sku+".webp", buf, true)
			if err != nil {
				continue
			}

			// Actualizar registro media con nuevo checksum
			updatePayload := map[string]interface{}{
				"check_sum":  sourceChecksum,
				"updated_at": time.Now().UTC().Format(time.RFC3339),
				"updated_by": 1,
			}
			_, _, err = sb.Client.From("media").Update(updatePayload, "", "").Eq("media_id", fmt.Sprintf("%d", media.MediaID)).Execute()
			if err != nil {
				fmt.Printf("[%s] Error updating media: %v\n", product.Sku, err)
				continue
			}

			fmt.Printf("[%s] ✅ Image deleted and re-uploaded successfully\n", product.Sku)
			updated++

		} else {
			// ===== CASO 4: Checksum igual → sin cambios =====
			skipped++
		}
	}

	fmt.Printf("\n📊 Summary: %d new, %d updated, %d skipped (total: %d)\n", insertedCount, updated, skipped, len(products))

	fmt.Println("All products updated successfully")
}

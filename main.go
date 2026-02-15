package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
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

func convertToWebp(sku string) ([]byte, error) {
	// convert
	var buf bytes.Buffer
	isJPG := false

	file, err := os.Open(WATCH_FILE_PATH + "/" + sku + ".png")
	if err != nil {
		fmt.Println("Error opening file: ", err)
		// return nil, err
		isJPG = true
		file, err = os.Open(WATCH_FILE_PATH + "/" + sku + ".jpg")
		if err != nil {
			fmt.Println("Error opening file: ", err)
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
		fmt.Println(err)
		return nil, err
	}

	// Convert the image to WebP format
	err = webp.Encode(&buf, img, &webp.Options{Quality: 80})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Save the WebP image to a file
	err = ioutil.WriteFile(OUTPUT_FILE_PATH+"/"+sku+".webp", buf.Bytes(), 0644)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func preparePayload(product model.Product) (*bytes.Buffer, string) {
	// create buffer
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add the file to hold the form
	file, err := os.Open(OUTPUT_FILE_PATH + "/" + product.Sku + ".webp")
	if err != nil {
		fmt.Printf("[%s]: media not exist\n", product.Sku)
		return nil, ""
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		fmt.Println("Error creating form file: ", err)
		return nil, ""
	}

	_, err = io.Copy(part, file)
	if err != nil {
		fmt.Println("Error copying file: ", err)
		return nil, ""
	}

	// Add the other fields
	err = writer.WriteField("alt", product.Description)
	if err != nil {
		fmt.Println("Error writing field: ", err)
		return nil, ""
	}

	err = writer.WriteField("slug", product.Sku)
	if err != nil {
		fmt.Println("Error writing field: ", err)
		return nil, ""
	}

	err = writer.WriteField("table_name", "product")
	if err != nil {
		fmt.Println("Error writing field: ", err)
		return nil, ""
	}

	productID := strconv.Itoa(int(product.ProductID))

	err = writer.WriteField("column_id", productID)
	if err != nil {
		fmt.Println("Error writing field: ", err)
		return nil, ""
	}

	err = writer.Close()
	if err != nil {
		fmt.Println("Error closing writer: ", err)
		return nil, ""
	}

	return &b, writer.FormDataContentType()
}

// func uploadImage(image *bytes.Buffer, wHeader string) interface{} {
// 	req, err := http.NewRequest(http.MethodPost, API_BASE_URL+ENDPOINT_MEDIA, image)
// 	if err != nil {
// 		fmt.Println("Error creating request: ", err)
// 	}

// 	req.Header.Add("User-Agent", "ms-product-image")
// 	req.Header.Add("Content-Type", wHeader)
// 	// req.Header.Add("Authorization", fmt.Sprintf("%s %s", "Bearer", TOKEN))

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		fmt.Println("Error sending request: ", err)
// 		return nil
// 	}
// 	defer resp.Body.Close()

// 	// read response
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		fmt.Println("Error reading response: ", err)
// 		return nil
// 	}
// 	var data interface{}
// 	err = json.Unmarshal(body, &data)
// 	if err != nil {
// 		fmt.Println("Error unmarshalling response: ", err)
// 		return nil
// 	}

// 	if resp.StatusCode != 200 && resp.StatusCode != 201 {
// 		fmt.Println("Error uploading image: ", data.(map[string]interface{})["message"])
// 		return nil
// 	}

// 	fmt.Println(data)
// 	return data.(map[string]interface{})["data"]
// }

func uploadToBucket(filename string, data []byte) error {
	bucketPath := fmt.Sprintf("/products/assets/webp/%s", filename)
	mimeType := "image/webp" // Especificar el tipo MIME correcto

	errorMessage, err := client.Storage.UploadFile("media", bucketPath, bytes.NewReader(data), storage_go.FileOptions{
		ContentType: &mimeType,
	})
	if err != nil {
		fmt.Printf("Error uploading file to bucket: %s, %s\n", errorMessage, err)
		return err
	}
	fmt.Printf("File uploaded successfully to bucket: %s\n", bucketPath)
	return nil
}

func main() {
	sb = &database.Supabase{
		Host:     "localhost",
		User:     "postgres",
		Password: "postgres",
		Port:     "5432",
		Dbname:   "ecommerce",
	}

	sb.Open(client)

	// d, i, e := client.From("product").Select("*", "", false).Execute()
	// if e != nil {
	// 	fmt.Println(e)
	// }
	// fmt.Println(i, d)
	// Read All Products
	// supabase
	products, err := sb.GetProducts()
	if err != nil {
		log.Fatal(err)
	}

	for _, product := range products {
		// if err != nil {
		// 	fmt.Println(err)
		// }

		// logic
		media := sb.GetMedia(product.Sku)

		if media.MediaID == 0 {
			// convert to webp
			buf, err := convertToWebp(product.Sku)
			if err != nil {
				fmt.Println(err)
				continue
			}

			// buffer
			// payload, _ := preparePayload(product)
			// if payload == nil {
			// 	continue
			// }

			// Upload to bucket
			err = uploadToBucket(product.Sku+".webp", buf)
			if err != nil {
				continue
			}

			// sintax update product here
			payload := map[string]interface{}{
				"mime_type":  "webp",
				"url":        fmt.Sprintf("%s/%s.webp", SUPABASE_PUBLIC_MEDIA_URL, product.Sku),
				"alt":        product.Name,
				"slug":       product.Sku,
				"is_active":  true,
				"created_at": strings.Split(time.Now().UTC().String(), " +")[0],
				"created_by": 1,
			}
			inserted, _, err := sb.Client.From("media").Insert(payload, false, "", "", "").Execute()
			if err != nil {
				fmt.Println(err)
			}
			newMedia := []model.Media{}
			err = json.Unmarshal(inserted, &newMedia)
			if err != nil {
				fmt.Println(err)
			}

			// update product
			payload = map[string]interface{}{
				"image_id":   newMedia[0].MediaID,
				"updated_at": strings.Split(time.Now().UTC().String(), " +")[0],
				"updated_by": 1,
			}
			_, _, err = sb.Client.From("product").Update(payload, "", "").Eq("product_id", fmt.Sprintf("%d", product.ProductID)).Execute()
			if err != nil {
				fmt.Println(err)
			}

			fmt.Printf("Updated %s to product successfully\n", product.Sku)

		} else {
			// fmt.Println(product.Sku, "exist")
			continue
		}

	}
}

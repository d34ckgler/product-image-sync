package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/chai2010/webp"
	"github.com/d34ckgler/sync-product-image/database"
	"github.com/d34ckgler/sync-product-image/model"
	"github.com/joho/godotenv"
)

var supabase *database.Supabase
var MONGODB_URI, MONGODB_DATABASE, TOKEN, API_BASE_URL, PREFIX_AUTHORIZATION, API_ADMIN_USER, API_ADMIN_PASSWORD, ENDPOINT_USER_LOGIN, ENDPOINT_MEDIA, WATCH_FILE_PATH, OUTPUT_FILE_PATH string
var err error

func init() {
	err = godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading environment variables")
	}

	// Initialize environment variables values
	API_BASE_URL = os.Getenv("API_BASE_URL")
	PREFIX_AUTHORIZATION = os.Getenv("PREFIX_AUTHORIZATION")
	ENDPOINT_MEDIA = os.Getenv("ENDPOINT_MEDIA")
	ENDPOINT_USER_LOGIN = os.Getenv("ENDPOINT_USER_LOGIN")
	API_ADMIN_USER = os.Getenv("API_ADMIN_USER")
	API_ADMIN_PASSWORD = os.Getenv("API_ADMIN_PASSWORD")
	MONGODB_URI = os.Getenv("MONGODB_URI")
	MONGODB_DATABASE = os.Getenv("MONGODB_DATABASE")
	WATCH_FILE_PATH = os.Getenv("WATCH_FILE_PATH")
	OUTPUT_FILE_PATH = os.Getenv("OUTPUT_FILE_PATH")
	TOKEN = os.Getenv("TOKEN")

	// Supabase String Connection
	SUPABASE_HOST := os.Getenv("sp_host")
	SUPABASE_USER := os.Getenv("sp_user")
	SUPABASE_PWD := os.Getenv("sp_password")
	SUPABASE_PORT := os.Getenv("sp_port")
	SUPABASE_DB := os.Getenv("sp_dbname")

	supabase = &database.Supabase{Host: SUPABASE_HOST, User: SUPABASE_USER, Password: SUPABASE_PWD, Port: SUPABASE_PORT, Dbname: SUPABASE_DB}
	// supabase.Connect()
	_, err = supabase.InitPool()
	if err != nil {
		log.Fatal(err)
	}
	// end initialize environment variables
}

func fetch(method, path, payload string) map[string]interface{} {
	url := os.Getenv("API_BASE_URL") + path

	req, _ := http.NewRequest(method, url, bytes.NewBufferString(payload))

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "ms-stellar-depoprod")
	///////////////// add token to header /////////////////
	// authorization := fmt.Sprintf("%s %s", PREFIX_AUTHORIZATION, token)
	// req.Header.Add("Authorization", authorization)

	res, _ := http.DefaultClient.Do(req)

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	var jsonResponse map[string]interface{}
	json.Unmarshal(body, &jsonResponse)
	return jsonResponse
}

func convertToWebp(sku string) {
	// convert
	var buf bytes.Buffer

	file, err := os.Open(WATCH_FILE_PATH + "/" + sku + ".jpg")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	// Decode the JPG image
	img, err := jpeg.Decode(file)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Convert the image to WebP format
	err = webp.Encode(&buf, img, &webp.Options{Quality: 80})
	if err != nil {
		fmt.Println(err)
		return
	}

	// Save the WebP image to a file
	err = ioutil.WriteFile(OUTPUT_FILE_PATH+"/"+sku+".webp", buf.Bytes(), 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
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

func uploadImage(image *bytes.Buffer, wHeader string) interface{} {
	req, err := http.NewRequest(http.MethodPost, API_BASE_URL+ENDPOINT_MEDIA, image)
	if err != nil {
		fmt.Println("Error creating request: ", err)
	}

	req.Header.Add("User-Agent", "ms-product-image")
	req.Header.Add("Content-Type", wHeader)
	req.Header.Add("Authorization", fmt.Sprintf("%s %s", PREFIX_AUTHORIZATION, TOKEN))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return nil
	}
	defer resp.Body.Close()

	// read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response: ", err)
		return nil
	}
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error unmarshalling response: ", err)
		return nil
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		fmt.Println("Error uploading image: ", data.(map[string]interface{})["message"])
		return nil
	}

	fmt.Println(data)
	return data.(map[string]interface{})["data"]
}

func main() {
	defer supabase.Close()

	// Read All Products
	// supabase
	products, err := supabase.GetProducts()
	if err != nil {
		log.Fatal(err)
	}

	for _, product := range products {
		if err != nil {
			fmt.Println(err)
		}

		// logic
		media := supabase.GetMedia(product.Sku)

		if media.MediaID == 0 {
			// convert to webp
			convertToWebp(product.Sku)

			// buffer
			payload, wHeader := preparePayload(product)
			if payload == nil {
				continue
			}

			image := uploadImage(payload, wHeader)
			if image == nil {
				continue
			}
			fmt.Println(product.Sku, image.([]interface{})[0].(map[string]interface{})["image_id"].(uint))

			// sintax update product here
		} else {
			fmt.Println(product.Sku, "exist")
			continue
		}

	}
}

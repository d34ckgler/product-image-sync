package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/chai2010/webp"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mdb *mongo.Database
var token string
var PERCENT_DISCOUNT_STOCK float64
var MONGODB_URI, MONGODB_DATABASE, API_BASE_URL, PREFIX_AUTHORIZATION, API_ADMIN_USER, API_ADMIN_PASSWORD, ENDPOINT_USER_LOGIN, ENDPOINT_MEDIA, WATCH_FILE_PATH, OUTPUT_FILE_PATH string
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
	// end initialize environment variables
}

func CreateMongoConnection() *mongo.Database {
	clientOptions := options.Client().ApplyURI(MONGODB_URI)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB")
	return client.Database(MONGODB_DATABASE)
}

func fetch(method, path, payload string) map[string]interface{} {
	url := os.Getenv("API_BASE_URL") + path

	req, _ := http.NewRequest(method, url, bytes.NewBufferString(payload))

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "ms-stellar-depoprod")
	///////////////// add token to header /////////////////
	authorization := fmt.Sprintf("%s %s", PREFIX_AUTHORIZATION, token)
	req.Header.Add("Authorization", authorization)

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

func getToken() string {
	// payload := "{\"email\":\"dortiz@balystiendas.com\",\"password\":\"yitp.HBBM0KD\"}"
	payload := fmt.Sprintf("{\"email\":\"%s\",\"password\":\"%s\"}", API_ADMIN_USER, API_ADMIN_PASSWORD)
	jsonResponse := fetch("POST", ENDPOINT_USER_LOGIN, payload)

	return jsonResponse["token"].(string)
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

func preparePayload(sku string) (*bytes.Buffer, string) {
	// create buffer
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add the file to hold the form
	file, err := os.Open(OUTPUT_FILE_PATH + "/" + sku + ".webp")
	if err != nil {
		fmt.Printf("[%s]: media not exist\n", sku)
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
	err = writer.WriteField("alt", "product image")
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

func uploadImage(image *bytes.Buffer, wHeader string) map[string]interface{} {
	req, err := http.NewRequest(http.MethodPost, API_BASE_URL+ENDPOINT_MEDIA, image)
	if err != nil {
		fmt.Println("Error creating request: ", err)
	}

	req.Header.Add("User-Agent", "ms-stellar-depoprod")
	req.Header.Add("Content-Type", wHeader)
	req.Header.Add("Authorization", fmt.Sprintf("%s %s", PREFIX_AUTHORIZATION, token))

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
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error unmarshalling response: ", err)
		return nil
	}

	fmt.Println(data["message"])
	return data["doc"].(map[string]interface{})
}

func main() {
	mdb = CreateMongoConnection()
	collectionProduct := mdb.Collection("productos")
	token = getToken()

	// Read All Products
	cursor, err := collectionProduct.Find(context.TODO(), bson.M{})
	if err != nil {
		fmt.Println(err)
	}

	for cursor.Next(context.TODO()) {
		var product map[string]interface{}
		err = cursor.Decode(&product)
		if err != nil {
			fmt.Println(err)
		}

		// check if image exist
		collectionMedia := mdb.Collection("media")
		var data = bson.M{}
		result := collectionMedia.FindOne(context.TODO(), bson.D{
			{
				Key:   "filename",
				Value: fmt.Sprintf("%s.webp", product["sku"]),
			}})

		err := result.Decode(&data)
		if err != nil && err.Error() == "mongo: no documents in result" {
			// convert to webp
			convertToWebp(product["sku"].(string))

			// buffer
			payload, wHeader := preparePayload(product["sku"].(string))
			if payload == nil {
				continue
			}

			image := uploadImage(payload, wHeader)
			if image == nil {
				continue
			}
			fmt.Println(product["sku"], image["id"])

			// update product
			update := bson.M{
				"$set": bson.M{
					"images": []bson.D{
						{
							{"image", image["id"]},
							{"id", image["id"]},
						}},
				},
			}
			_, err = collectionProduct.UpdateOne(context.TODO(), bson.M{"sku": product["sku"]}, update)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			fmt.Println(product["sku"], "exist")
			continue
		}

	}
}

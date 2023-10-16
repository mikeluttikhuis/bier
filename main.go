package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"text/template"
	"time"

	"gopkg.in/yaml.v2"
)

type Stores struct {
	FullName string `yaml:"fullName"`
	StoreId  int    `yaml:"storeId"`
}

type Brands struct {
	FullName string `yaml:"fullName"`
	BrandId  int    `yaml:"brandId"`
}

type Http struct {
	CacheTimeout int `yaml:"cacheTimeout"`
	ListenPort   int `yaml:"listenPort"`
}
type Config struct {
	Stores []Stores `yaml:"stores"`
	Brands []Brands `yaml:"brands"`
	Http   Http     `yaml:"http"`
}

func fetch(config Config) []map[string]string {
	// Create new HTTP requests
	req, err := http.NewRequest("GET", "https://www.biernet.nl/extra/app/V3_3.3.4/aanbieding.php", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		panic(err)
	}

	// Set the iOS User Agent
	req.Header.Set("User-Agent", "nl.Biernet.iOS.app/V3")

	// Create the client and make the request
	client := &http.Client{}
	client.Timeout = 3 * time.Second
	resp_discount, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		panic(err)
	}
	defer resp_discount.Body.Close()

	// Parse response as JSON
	var data_discount interface{}
	err = json.NewDecoder(resp_discount.Body).Decode(&data_discount)
	if err != nil {
		panic(err)
	}
	discounts := make([]map[string]string, 0)

	for _, discount := range data_discount.([]interface{}) {
		for _, brand := range config.Brands {
			for _, store := range config.Stores {
				store_uid, _ := strconv.Atoi(discount.(map[string]interface{})["winkel_uid"].(string))
				brand_uid, _ := strconv.Atoi(discount.(map[string]interface{})["soort_uid"].(string))
				if store_uid == store.StoreId && brand_uid == brand.BrandId {
					discounts = append(discounts, map[string]string{
						"fromPrice":     discount.(map[string]interface{})["vanprijs"].(string),
						"discountPrice": discount.(map[string]interface{})["voorprijs"].(string),
						"startDate":     discount.(map[string]interface{})["begindatum"].(string),
						"endDate":       discount.(map[string]interface{})["einddatum"].(string),
						"amount":        discount.(map[string]interface{})["aantal"].(string),
						"storeName":     store.FullName,
						"brandName":     brand.FullName,
					})
				}
			}
		}
	}
	return discounts
}

func serve(discounts []map[string]string, config Config) {
	// Define a mutex to synchronize access to the cached data.
	var mutex sync.Mutex

	// Initialize the cached data with an empty slice and an expiration time of 0.
	var expiration time.Time

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(404)
			w.Write([]byte(`Not Found`))
			return
		}

		// Lock the mutex to prevent concurrent access to the cached data.
		mutex.Lock()
		defer mutex.Unlock()

		// Check if cached data is still valid.
		if time.Now().Before(expiration) {
			tmpl, err := template.ParseFiles("template.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// If the cached data is still valid, write it to the response.
			err = tmpl.Execute(w, discounts)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			// If the cached data is no longer valid, fetch the data from the source.
			// TODO: Do some error handling if fetch() errors, then just serve from cache.
			fresh_discounts := fetch(config)

			// Update the cached data with the new data and expiration time.
			expiration = time.Now().Add(time.Second * time.Duration(config.Http.CacheTimeout))
			discounts = fresh_discounts // Update the cached data.

			// Write the new data to the response.
			tmpl, err := template.ParseFiles("template.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = tmpl.Execute(w, fresh_discounts)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	})

	// Serve the webserver on the configured port
	http.ListenAndServe(fmt.Sprintf(":%v", config.Http.ListenPort), nil)

}
func main() {
	// Read YAML file
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Parse YAML data into Config struct
	config := Config{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Fetch data from Biernet API
	discounts := fetch(config)

	// Start the webserver and serve the discounts
	serve(discounts, config)
}

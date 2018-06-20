package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-restit/lzjson"
	"github.com/jinzhu/gorm"
)

const (
	australiangeographic = "ausGeo"
	barnesandnoble       = "barnesNoble"
	bestbuycanada        = "bestBuyCanada"
	bestbuyusa           = "bestBuyUS"
	indigo               = "indigo"
	jbhifi               = "jbHiFi"
	target               = "target"
	thesource            = "theSource"
	toysruscanada        = "truCanada"
	toysrususa           = "truUS"
	walmart              = "walmart"
)

func getBrands() []string {
	return []string{
		australiangeographic,
		barnesandnoble,
		bestbuycanada,
		bestbuyusa,
		indigo,
		jbhifi,
		target,
		thesource,
		toysruscanada,
		toysrususa,
		walmart,
	}

}

// caProvinceCodes returns Canadian province names (and their typos) to their 2 digit codes
func caProvinceCodes() map[string]string {
	return map[string]string{
		"alberta":                   "AB",
		"british columbia":          "BC",
		"manitoba":                  "MB",
		"new brunswick":             "NB",
		"newfoundland and labrador": "NL",
		"newfoundland":              "NL",
		"northwest territories":     "NT",
		"nova scotia":               "NS",
		"novia scotia":              "NS",
		"nunavut":                   "NU",
		"ontario":                   "ON",
		"prince edward island":      "PE",
		"pei":          "PE",
		"quebec":       "QC",
		"saskatchewan": "SK",
		"yukon":        "YT",
	}
}

func nstr(input string) sql.NullString {
	cleaned := cleanString(input)
	if cleaned == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{String: cleaned, Valid: true}
}

func transform(brand string) (Stores, error) {
	filename := fmt.Sprintf("./data/initial/%s.json", brand)
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var dirties []map[string]interface{}
	var cleanse []map[string]string
	err = json.NewDecoder(bytes.NewReader(f)).Decode(&dirties)
	if err != nil {
		return nil, err
	}

	for _, dirty := range dirties {
		cleaned := cleanMapFields(dirty)
		cleanse = append(cleanse, cleaned)
	}

	cleanedBytes, err := json.Marshal(cleanse)
	if err != nil {
		return nil, err
	}

	stores := Stores{}
	results := lzjson.Decode(bytes.NewReader(cleanedBytes))
	for i := 0; i < results.Len(); i++ {
		result := results.GetN(i)
		store, err := transformRecord(brand, result)
		if err != nil {
			return nil, err
		}

		stores = append(stores, store)
	}

	return stores, nil
}

func transformRecord(brand string, result lzjson.Node) (*Store, error) {
	store := NewStore()
	switch brand {
	case australiangeographic:
		address1 := result.Get("ADDRESS1").String()
		suburb := result.Get("SUBURB").String()
		if address1 == "CLOSED" || suburb == "CLOSED" {
			log.Printf("Brand: %s store is closed", brand)
			store.Error = nstr("store is closed")
		}

		var (
			city string
			zip  string
		)
		splitSuburb := []string{}
		for _, bit := range strings.Split(suburb, " ") {
			if bit == "" {
				continue
			}
			splitSuburb = append(splitSuburb, bit)
		}

		if len(splitSuburb) > 0 {
			if _, err := strconv.ParseInt(splitSuburb[len(splitSuburb)-1], 10, 64); err == nil {
				zip = splitSuburb[len(splitSuburb)-1]
				splitSuburb = splitSuburb[:len(splitSuburb)-1]
			}
			if splitSuburb[len(splitSuburb)-1] == result.Get("STATE").String() {
				splitSuburb = splitSuburb[:len(splitSuburb)-1]
			}

			if len(splitSuburb) > 0 {
				city = strings.Join(splitSuburb, " ")
			}
		}

		if result.Get("BRAND").String() == "Co-op" {
			store.Brand = nstr("The Co-op")
		} else {
			store.Brand = nstr("Australian Geographic")
		}

		store.Name = nstr(result.Get("STORE").String())
		store.Address = nstr(fmt.Sprintf("%s %s", result.Get("ADDRESS1"), result.Get("ADDRESS2")))
		store.City = nstr(city)
		store.State = nstr(result.Get("STATE").String())
		store.Zip = nstr(zip)
		if phone := result.Get("PHONE").String(); phone != "TBA" {
			store.Phone = nstr(phone)
		}
		if result.Get("BRAND").String() == "Co-op" {
			store.Website = nstr("https://www.coop.com.au")
		} else {
			store.Website = nstr("http://www.australiangeographic.com.au")
		}

	case barnesandnoble:
		store.Brand = nstr("Barnes & Noble")
		store.Name = nstr(result.Get("BUSINESS").String())
		store.Address = nstr(result.Get("ADDRESS").String())
		store.City = nstr(result.Get("CITY").String())
		store.State = nstr(result.Get("STATE").String())
		store.Zip = nstr(result.Get("ZIP").String())
		store.Website = nstr("https://www.barnesandnoble.com/s/kano+computer?_requestid=462753")

	case bestbuycanada:
		store.Brand = nstr("Best BUY")
		store.Name = nstr(result.Get("STORENAME").String())
		store.Address = nstr(result.Get("ADDRESS").String())
		store.City = nstr(result.Get("CITY").String())
		store.State = nstr(result.Get("STATE").String())
		store.Zip = nstr(result.Get("ZIP").String())
		store.Website = nstr("https://www.bestbuy.ca/en-CA/home.aspx")

	case bestbuyusa:
		doesNotStockID := "#N/A"
		stockComputerKit := result.Get("COMPUTERKIT").String()
		stockPixelKit := result.Get("PIXELKIT").String()
		stockMSK := result.Get("MSK").String()
		if stockComputerKit != doesNotStockID || stockPixelKit != doesNotStockID || stockMSK != doesNotStockID {
			store.Brand = nstr("Best Buy")
			store.Name = nstr(result.Get("LOCATIONNAME").String())
			store.Address = nstr(result.Get("ADDRESS1").String())
			if address2 := cleanString(result.Get("ADDRESS2").String()); address2 != "" {
				store.Address = nstr(store.Address.String + ", " + address2)
			}
			store.City = nstr(result.Get("CITY").String())
			store.State = nstr(result.Get("STATE").String())
			store.Zip = nstr(result.Get("ZIPCODE").String())
			store.Phone = nstr(result.Get("TELEPHONENBR").String())
			store.Website = nstr("https://www.bestbuy.com/site/searchpage.jsp?cp=1&searchType=search&st=kano&_dyncharset=UTF-8&id=pcat17071&type=page&sc=Global&nrp=&sp=&qp=brand_facet%3DBrand~Kano&list=n&af=true&iht=y&usc=All%20Categories&ks=960&keys=keys")
		}

	case indigo:
		store.Brand = nstr("Indigo")
		store.Name = nstr(result.Get("STORENAME").String())
		store.Address = nstr(result.Get("STOREADDRESS").String())
		store.City = nstr(result.Get("STORECITY").String())
		store.State = nstr(result.Get("STOREPROVINCE").String())
		store.Zip = nstr(result.Get("STOREPC").String())
		store.Website = nstr("https://www.chapters.indigo.ca/en-ca/home/search/?keywords=kano&internal=1#facetIds=4294941581&page=0&pid=506040280099&sortDirection=&sortKey=")

	case jbhifi:
		store.Brand = nstr("JB Hi-Fi")
		store.Name = nstr(result.Get("STORENAME").String())
		store.Address = nstr(result.Get("ADDRESS").String())
		store.City = nstr(result.Get("SUBURB").String())
		store.State = nstr(result.Get("STATE").String())
		store.Zip = nstr(fmt.Sprintf("%d", result.Get("POSTCODE").Int()))
		store.Phone = nstr(result.Get("PHONE").String())
		store.Website = nstr("https://www.jbhifi.com.au")

	case target:
		store.Brand = nstr("Target")
		store.Name = nstr(result.Get("NAME").String())
		store.Address = nstr(result.Get("ADDRESS").String())
		store.City = nstr(result.Get("CITY").String())
		store.State = nstr(result.Get("STATE").String())
		store.Zip = nstr(result.Get("ZIP").String())
		store.Website = nstr("https://www.target.com")

	case thesource:
		store.Brand = nstr("The Source")
		store.Name = nstr(result.Get("LOCATIONNAME").String())
		store.Address = nstr(result.Get("ADDRESS").String())
		store.City = nstr(result.Get("CITY").String())
		store.State = nstr(result.Get("PROVINCE").String())
		store.Zip = nstr(result.Get("POSTCODE").String())
		store.Website = nstr("https://www.thesource.ca/en-ca/brands/kano/c/kano")

	case toysruscanada:
		province := cleanString(strings.ToLower(result.Get("STATE").String()))
		provinceCode, ok := caProvinceCodes()[province]
		if !ok {
			store.Error = nstr(fmt.Sprintf("invalid ca province code: %s", province))
		}
		store.Brand = nstr("Toys \"R\" Us")
		store.Name = nstr(result.Get("STORENAME").String())
		store.Address = nstr(result.Get("ADDRESS").String())
		store.City = nstr(result.Get("CITY").String())
		store.State = nstr(provinceCode)
		store.Zip = nstr(result.Get("ZIP").String())
		store.Website = nstr("http://www.toysrus.ca/home/index.jsp")

	case toysrususa:
		store.Brand = nstr("Toys \"R\" Us")
		store.Name = nstr(result.Get("NAME").String())
		store.Address = nstr(result.Get("ADDRESS1").String())
		if address2 := cleanString(result.Get("ADDRESS2").String()); address2 != "" {
			store.Address = nstr(store.Address.String + ", " + address2)
		}
		store.City = nstr(result.Get("CITY").String())
		store.State = nstr(result.Get("STATE").String())
		store.Zip = nstr(result.Get("ZIP").String())
		store.Website = nstr("https://www.toysrus.com/family?categoryid=7367219")

	case walmart:
		store.Brand = nstr("Walmart")
		store.Name = nstr(result.Get("STORENAME").String())
		store.Address = nstr(result.Get("ADDRESS").String())
		store.City = nstr(result.Get("CITY").String())
		store.State = nstr(result.Get("STATE").String())
		store.Zip = nstr(result.Get("ZIP").String())
		store.Website = nstr("https://www.walmart.com")

	default:
		return nil, errors.New("no matching brand found")
	}
	store.cleanse()
	store.Key = store.HashKey()
	return store, nil
}

// initBrands populate all existing brand')s JSON and upsert into database
func initBrands(db *gorm.DB) error {
	time.Sleep(time.Second * 1)
	brands := getBrands()
	results := make(chan Stores, 10)

	go func() {
		for {
			stores, more := <-results
			if more {
				log.Printf("Loading %d stores...\n", len(stores))
				err := SaveStoresInTransaction(stores, db)
				if err != nil {
					log.Println("Error saving stores: ", err.Error())
					continue
				}
				log.Printf("%d stores saved successfully\n", len(stores))
				time.Sleep(apiGeocodeWait)
			} else {
				log.Println("All stores handled")
				return
			}
		}
	}()

	for _, brand := range brands {
		transformed, err := transform(brand)
		if err != nil {
			log.Printf("Error parsing %s: %s", brand, err.Error())
			return err
		}
		log.Printf("Parsing & transforming brand: %s", brand)
		results <- transformed
	}
	close(results)

	return nil
}

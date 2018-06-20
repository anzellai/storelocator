package main

import (
	"bytes"
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

// GetDB returns database handle
func GetDB() *gorm.DB {
	db, err := gorm.Open("sqlite3", "./stores.db")
	if err != nil {
		panic(err)
	}
	//return db.LogMode(true)
	return db
}

// LookupFields return all lookup-able field names
func LookupFields() []string {
	return []string{
		"brand",
		"name",
		"address",
		"city",
		"state",
		"zip",
		"phone",
		"website",
		"error",
	}
}

// StoreByKey return single store with given key
func StoreByKey(key string) (*Store, error) {
	db := GetDB()
	var store Store
	err := db.Where("key = ?", key).First(&store).Error
	return &store, err
}

// LookupStores query database and return matching stores
func LookupStores(keyword string) (Stores, error) {
	var stores Stores
	db := GetDB()
	if keyword == "error" {
		err := db.Where("error IS NOT ?", nil).Find(&stores).Error
		return stores, err
	}
	keywords := []interface{}{}
	lookupFields := LookupFields()
	for range lookupFields {
		keywords = append(keywords, "%"+keyword+"%")
	}
	query := strings.Join(lookupFields, " LIKE ? OR ") + " LIKE ?"
	err := db.Where(query, keywords...).Find(&stores).Error
	return stores, err
}

func synopsis(input string) string {
	if len(input) > 40 {
		return input[:40] + "..."
	}
	return input
}

func cleanMapField(input string) string {
	return strings.ToUpper(input)
}

func cleanString(input string) string {
	return strings.Join(strings.Fields(input), " ")
}

func forceString(input interface{}) string {
	switch input := input.(type) {
	case int:
		return fmt.Sprintf("%d", input)
	case float64:
		return fmt.Sprintf("%f", input)
	case bool:
		return fmt.Sprintf("%t", input)
	default:
		return string(input.(string))
	}
}

func cleanMapFields(input map[string]interface{}) map[string]string {
	output := make(map[string]string)
	for key, value := range input {
		val := forceString(value)
		output[cleanMapField(key)] = val
	}
	return output
}

func cleanNullableString(input sql.NullString) sql.NullString {
	if input.Valid {
		cleanedString := cleanString(input.String)
		return sql.NullString{String: cleanedString, Valid: cleanedString != ""}
	}
	return input
}

// Location is the embed struct for Location
type Location struct {
	gorm.Model
	StoreKey   string  `json:"-" gorm:"column:store_key"`
	Key        string  `json:"-" gorm:"unique"`
	GeoAddress string  `json:"-"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
}

// NewLocation returns new instance of Geopoint
func NewLocation(s *Store, lat, lng float64) Location {
	return Location{
		Key: s.Key,
		Lat: lat,
		Lng: lng,
	}
}

// Store struct for cleansed result
type Store struct {
	gorm.Model
	Key      string         `json:"_key_" gorm:"primary_key;unique_index;column:key"`
	Brand    sql.NullString `json:"brand"`
	Name     sql.NullString `json:"name"`
	Address  sql.NullString `json:"address"`
	City     sql.NullString `json:"city"`
	State    sql.NullString `json:"state"`
	Zip      sql.NullString `json:"zip"`
	Phone    sql.NullString `json:"phone"`
	Website  sql.NullString `json:"website"`
	Location Location       `json:"location,omitempty" gorm:"foreignkey:StoreKey"`
	Error    sql.NullString `json:"error,omitempty"`
}

// Stores alias to slice of Store
type Stores = []*Store

type exportedLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type exportedStore struct {
	Key      string           `json:"_key_"`
	Brand    *string          `json:"brand"`
	Name     *string          `json:"name"`
	Address  *string          `json:"address"`
	City     *string          `json:"city"`
	State    *string          `json:"state"`
	Zip      *string          `json:"zip"`
	Phone    *string          `json:"phone"`
	Website  *string          `json:"website"`
	Location exportedLocation `json:"location"`
}

// String interface
func (s *Store) String() string {
	return fmt.Sprintf(
		"<%s: %s | %s | %s {%f, %f} %s>",
		s.Key,
		synopsis(s.Brand.String),
		synopsis(s.Name.String),
		synopsis(s.Address.String),
		//synopsis(s.City.String),
		//synopsis(s.State.String),
		//synopsis(s.Zip.String),
		//synopsis(s.Phone.String),
		//synopsis(s.Website.String),
		s.Location.Lat,
		s.Location.Lng,
		synopsis(s.Error.String),
	)
}

// GetAddress returns string representation of address components
func (s *Store) GetAddress() string {
	geoAddresses := []string{}
	if s.Address.Valid {
		geoAddresses = append(geoAddresses, s.Address.String)
	}
	if s.City.Valid {
		geoAddresses = append(geoAddresses, s.City.String)
	}
	if s.State.Valid {
		geoAddresses = append(geoAddresses, s.State.String)
	}
	geoAddress := strings.Join(geoAddresses, ", ")
	return geoAddress
}

// SaveStoresInTransaction saves the stores slice in transaction
func SaveStoresInTransaction(stores Stores, db *gorm.DB) (err error) {
	for _, store := range stores {
		if err := SaveStoreInTransaction(store, db); err != nil {
			return err
		}
	}
	return nil
}

// SaveStoreInTransaction saves the stores slice in transaction
func SaveStoreInTransaction(store *Store, db *gorm.DB) (err error) {
	if err := db.Where(Store{Key: store.Key}).FirstOrCreate(store).Error; err != nil {
		return err
	}
	return nil
}

// HashKey returns or generates new sha1 hash based on struct values
func (s *Store) HashKey() string {
	key := s.Key
	if key != "" {
		return key
	}

	s.cleanse()

	hash := sha1.New()
	hash.Write([]byte(s.Brand.String))
	hash.Write([]byte(s.Name.String))
	hash.Write([]byte(s.Address.String))
	hash.Write([]byte(s.City.String))
	hash.Write([]byte(s.State.String))
	hash.Write([]byte(s.Zip.String))
	hash.Write([]byte(s.Phone.String))
	hash.Write([]byte(s.Website.String))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// NewStore return new instance of Store
func NewStore() *Store {
	return &Store{}
}

// jsonMarshal marshals struct to non-escaped literal
func jsonMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "    ")
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

// StoresToJSON renders stores into JSON bytes
func StoresToJSON(stores Stores) ([]byte, error) {
	db := GetDB()
	var exported []exportedStore
	for _, store := range stores {
		err := db.Model(store).Association("location").Find(&store.Location).Error
		if err != nil {
			//fmt.Printf("Error fetching location from store: %v\n", err)
			continue
		}
		keyvalues := exportedStore{Key: store.Key}
		if store.Brand.Valid {
			keyvalues.Brand = &store.Brand.String
		}
		if store.Name.Valid {
			keyvalues.Name = &store.Name.String
		}
		if store.Address.Valid {
			keyvalues.Address = &store.Address.String
		}
		if store.City.Valid {
			keyvalues.City = &store.City.String
		}
		if store.State.Valid {
			keyvalues.State = &store.State.String
		}
		if store.Zip.Valid {
			keyvalues.Zip = &store.Zip.String
		}
		if store.Phone.Valid {
			keyvalues.Phone = &store.Phone.String
		}
		if store.Website.Valid {
			keyvalues.Website = &store.Website.String
		}
		keyvalues.Location = exportedLocation{
			Lat: store.Location.Lat,
			Lng: store.Location.Lng,
		}
		exported = append(exported, keyvalues)
	}
	return jsonMarshal(exported)
}

// cleanse returns a new and cleansed instance of Store, not mutating original record
func (s *Store) cleanse() {
	s.Brand = cleanNullableString(s.Brand)
	s.Name = cleanNullableString(s.Name)
	s.Address = cleanNullableString(s.Address)
	s.City = cleanNullableString(s.City)
	s.State = cleanNullableString(s.State)
	s.Zip = cleanNullableString(s.Zip)
	s.Phone = cleanNullableString(s.Phone)
	s.Website = cleanNullableString(s.Website)
}

// sortByKey sorts the store slice in place by hash key value
func sortByKey(stores Stores) {
	n := len(stores)
	for i := 1; i < n; i++ {
		j := i
		for j > 0 {
			if stores[j-1].Key > stores[j].Key {
				stores[j-1], stores[j] = stores[j], stores[j-1]
			}
			j = j - 1
		}
	}
}

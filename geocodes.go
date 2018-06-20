package main

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"googlemaps.github.io/maps"
)

const (
	apiGeocodeKey      = "AIzaSyDq0YJCg0bJv6O8ZZ5qSVu9H4qehEi-SBU"
	apiGeocodeWait     = time.Millisecond * 50
	apiGeocodeEndpoint = "https://maps.googleapis.com/maps/api/geocode/json"
)

// getGeocode call maps API and return the first result
func getGeocode(s *Store) (Location, error) {
	//time.Sleep(apiWait)
	var location Location

	client, err := maps.NewClient(maps.WithAPIKey(apiGeocodeKey))
	if err != nil {
		return location, err
	}

	geoAddress := s.GetAddress()

	request := &maps.GeocodingRequest{
		Address: geoAddress,
	}

	ctx := context.Background()
	results, err := client.Geocode(ctx, request)
	if err != nil {
		return location, err
	}
	if len(results) == 0 {
		err = errors.New("returned no results")
		return location, err
	}
	location = NewLocation(s, results[0].Geometry.Location.Lat, results[0].Geometry.Location.Lng)

	return location, nil
}

// initGeocode populate all missing geocode locations from database
func initGeocode(db *gorm.DB) error {
	time.Sleep(time.Second * 1)

	var stores Stores
	results := make(chan *Store, 10)

	if err := db.Find(&stores).Error; err != nil {
		return err
	}

	go func() {
		for {
			store, more := <-results
			if more {
				location, err := getGeocode(store)
				if err != nil {
					log.Println("Error parsing store address: ", err.Error())
					store.Error = nstr("location error: " + err.Error())
				} else {
					log.Printf("Location received: %+v", location)
					store.Location = location
					store.Error = nstr("")
				}

				err = db.Model(store).Save(store).Error
				if err != nil {
					log.Printf("Error saving store: %s", err.Error())
				}

				log.Printf("Store's with location: %+v", store)
				time.Sleep(apiGeocodeWait)
			} else {
				log.Println("All store locations handled")
				return
			}
		}
	}()

	for _, store := range stores {
		err := db.Model(store).Association("location").Find(&store.Location).Error
		if err != nil || store.Error.Valid && strings.HasPrefix(store.Error.String, "location error:") {
			results <- store
		}
	}
	close(results)

	return nil
}

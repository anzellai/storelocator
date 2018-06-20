package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
)

const (
	actionQuit     = "Quit"
	actionLookup   = "Lookup store(s) from database"
	actionExport   = "Export cleansed and non-error store(s) in JSON format"
	actionRunAll   = "Populate initial brand + geocode data"
	actionRunBrand = "Populate/re-run only brand data"
	actionRunGeo   = "Populate/re-run only missing geocode data"
)

func populateData(args arguments) {
	db := GetDB()

	db.AutoMigrate(&Store{})
	db.AutoMigrate(&Location{})

	if args.seedPtr {
		log.Println("Populating brands data...")
		if err := initBrands(db); err != nil {
			log.Fatal(err)
		}
	}

	if args.geoPtr {
		log.Println("Populating missing geocode data...")
		if err := initGeocode(db); err != nil {
			log.Fatal(err)
		}
	}
}

func runPrompt() {
	for {
		_, result, err := makeSelect(
			"Please select an Action to execute",
			[]string{
				actionQuit,
				actionLookup,
				actionExport,
				actionRunAll,
				actionRunBrand,
				actionRunGeo,
			},
		)

		if err != nil {
			return
		}

		switch result {
		case actionLookup:
			lookupPrompt()
		case actionExport:
			exportPrompt()
		case actionRunAll:
			populateData(arguments{seedPtr: true, geoPtr: true})
		case actionRunBrand:
			populateData(arguments{seedPtr: true, geoPtr: false})
		case actionRunGeo:
			populateData(arguments{seedPtr: false, geoPtr: true})
		default:
			fmt.Println("Quitting...")
			return
		}

		fmt.Printf("\n\nAction %q completed\n", result)
	}
}

func exportPrompt() {
	var stores Stores
	db := GetDB()
	err := db.Where("error IS ?", nil).Find(&stores).Error
	if err != nil {
		fmt.Printf("Error fetching stores with non-error: %v\n", err)
		return
	}

	sortByKey(stores)

	exportedBytes, err := StoresToJSON(stores)
	if err != nil {
		fmt.Printf("Error marshalling stores into JSON: %v\n", err)
		return
	}
	fmt.Printf("Rendering %d stores in JSON format...\n", len(stores))
	fmt.Println(string(exportedBytes))
	fmt.Printf("Rendered %d stores in JSON format...\n", len(stores))

	exportedPath := "./data/results"
	exportedFile := filepath.Join(exportedPath, "stores.json")
	if _, err := os.Stat(exportedPath); os.IsNotExist(err) {
		os.Mkdir(exportedPath, os.ModePerm)
	}
	err = ioutil.WriteFile(exportedFile, exportedBytes, 0644)
	if err != nil {
		fmt.Printf("Error writing exported JSON to %s: %v\n", exportedFile, err)
		return
	}
	fmt.Printf("Exported JSON to file: %s\n", exportedFile)
}

func makeSelect(label string, items []string) (int, string, error) {
	prompt := promptui.Select{
		Label: label,
		Items: items,
	}
	idx, result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return idx, "", err
	}
	fmt.Printf("You choose %q\n", result)
	return idx, result, nil
}

func makePrompt(label string) (string, error) {
	prompt := promptui.Prompt{
		Label: label,
	}
	result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return "", err
	}
	fmt.Printf("You enter %q\n", result)
	return result, nil
}

func lookupPrompt() {
	for {
		keyword, err := makePrompt("Please enter keyword for lookup (type 'error' for stores with error message)")
		if keyword == "" || err != nil {
			break
		}

		stores, err := LookupStores(keyword)
		if err != nil {
			fmt.Printf("Error looking up keyword: %v\n", err)
			continue
		}

		storeSelections := []string{}
		for _, store := range stores {
			storeSelections = append(storeSelections, store.String())
		}

		_, store, err := makeSelect(
			"Please select Store for further action",
			storeSelections,
		)
		if err != nil {
			fmt.Printf("Error selecting store: %v\n", err)
			continue
		}

		storeKey := strings.Split(store[1:], ":")[0]
		fmt.Printf("Store Key: %s selected", storeKey)

		err = editPrompt(storeKey)
		if err != nil {
			fmt.Printf("Error transiting to edit prompt: %v", err)
			return
		}
	}
}

func editPrompt(storeKey string) error {
	for {
		store, err := StoreByKey(storeKey)
		if err != nil {
			fmt.Printf("Error getting store by key: %v\n", err)
			return err
		}

		lookupFields := LookupFields()
		keyvalues := map[string]string{
			"brand":   store.Brand.String,
			"name":    store.Name.String,
			"address": store.Address.String,
			"city":    store.City.String,
			"state":   store.State.String,
			"zip":     store.Zip.String,
			"phone":   store.Phone.String,
			"website": store.Website.String,
			"error":   store.Error.String,
		}

		selection := []string{}
		for _, lookupField := range lookupFields {
			value := keyvalues[lookupField]
			selection = append(selection, lookupField+": "+value)
		}

		_, result, err := makeSelect(
			"Please select which field you want to edit",
			selection,
		)

		if err != nil {
			return err
		}

		selected := strings.Split(result, ":")
		selectedKey := selected[0]

		promptMessage := fmt.Sprintf("Please enter the new value (current %s)", selected)
		changed, err := makePrompt(promptMessage)
		if err != nil {
			return err
		}

		db := GetDB()
		err = db.Model(store).Update(selectedKey, changed).Error
		if err != nil {
			fmt.Printf("Error updating store changes: %v\n", err)
			return err
		}
		fmt.Println("Update store successfully")
	}

}

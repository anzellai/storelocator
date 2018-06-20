package main

import (
	"flag"
	"fmt"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var (
	// VERSION from build flag
	VERSION string
	// COMMIT from build flag
	COMMIT string
	// BRANCH from build flag
	BRANCH string
)

type arguments struct {
	seedPtr bool
	geoPtr  bool
}

func parseFlags() arguments {
	args := arguments{}

	flag.BoolVar(&args.seedPtr, "seed", false, "populate seeds data")
	flag.BoolVar(&args.geoPtr, "geo", false, "populate all missing geocode")
	flag.Parse()

	return args
}

func main() {
	fmt.Println("--------------------------------------------------")
	programMeta := fmt.Sprintf(
		" Welcome to StoreLocator v%s\n\n Commit: %s\n Branch: %s\n",
		VERSION, COMMIT, BRANCH,
	)
	fmt.Println(programMeta)
	fmt.Println("--------------------------------------------------")

	args := parseFlags()
	if args.seedPtr || args.geoPtr {
		populateData(args)
	} else {
		runPrompt()
	}
}

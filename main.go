package main

import (
	"flag"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
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
	args := parseFlags()
	if args.seedPtr || args.geoPtr {
		populateData(args)
	} else {
		runPrompt()
	}
}

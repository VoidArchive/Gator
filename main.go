package main

import (
	"fmt"
	"log"

	"github.com/voidarchive/Gator/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal("Error reading config:", err)
	}
	if err := cfg.SetUser("anish"); err != nil {
		log.Fatal("Error setting user:", err)
	}

	cfg, err = config.Read()
	if err != nil {
		log.Fatal("Error reading updated config:", err)
	}
	fmt.Printf("Config: %+v\n", cfg)
}

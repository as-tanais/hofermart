package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/as-tanais/hofermart/internal/config"
)

func main() {
	runAddr := flag.String("a", "", "Server address :8080")
	dbURI := flag.String("d", "", "Database DSN")
	flag.Parse()

	cfg, err := config.LoadAccrualConfig(*runAddr, *dbURI)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg)
}

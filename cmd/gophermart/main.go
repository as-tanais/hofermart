package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/as-tanais/hofermart/internal/config"
)

func main() {

	addr := flag.String("a", "", "Server address")
	dsn := flag.String("d", "", "DSN")
	accrualAddr := flag.String("r", "", "Accrual system address (e.g. http://accrual:8080)")

	flag.Parse()

	cfg, err := config.LoadGophermartConfig(*addr, *dsn, *accrualAddr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg)
}

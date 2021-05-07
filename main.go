package main

import (
	"fmt"
	"os"

	"github.com/lendefi/api/ldfi"
	"github.com/lendefi/api/server"
)

func main() {
	client, err := ldfi.NewClientFromEnv()
	if err != nil {
		fmt.Println("Can't create client:", err.Error())
		os.Exit(1)
	}

	srv, err := server.NewServerFromEnv(client)
	if err != nil {
		fmt.Println("Can't create http server:", err.Error())
		os.Exit(1)
	}

	if err = srv.Serve(); err != nil {
		fmt.Println("Can't listen http server:", err.Error())
		os.Exit(1)
	}
}

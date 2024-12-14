package main

import (
	"fmt"

	"github.com/mpcwallet/service"
	"github.com/sirupsen/logrus"
)

func main() {
	var log = logrus.NewEntry(logrus.New())
	s, err := service.NewWalletMPCService(service.ServerOpts{ListenAddr: ":8080", Log: log})
	if err != nil {
		fmt.Printf("Failed to start server: %s", err)
	}
	fmt.Println("Server started")
	s.StartHTTPServer()
}

package main

import (
	"github.com/mpcwallet/service"
	"github.com/sirupsen/logrus"
)

func main() {
	var log = logrus.NewEntry(logrus.New())
	s, err := service.NewWalletMPCService(service.ServerOpts{ListenAddr: ":8080", Log: log})
	if err != nil {
		log.Infof("Failed to start server: %s", err)
	}
	log.Info("Server started")
	s.StartHTTPServer()
}

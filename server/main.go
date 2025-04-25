package main

import (
	"log"
	"os"

	. "github.com/seatsurfing/seatsurfing/server/app"
	. "github.com/seatsurfing/seatsurfing/server/messaging"
	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/util"
)

func main() {
	log.Println("Starting...")
	log.Println("Seatsurfing Backend Version " + GetProductVersion())
	db := GetDatabase()
	a := GetApp()
	a.InitializeDatabases()
	a.InitializeDefaultOrg()
	a.InitializePlugins()
	a.InitializeRouter()
	a.InitializeTimers()
	log.Println("Need to init slack client")
	s := InitializeSlackClient()
	// webserver
	a.Run()
	db.Close()

	// Shutdown sequence
	s.Shutdown() // Gracefully stop Slack client (waits for goroutines)
	log.Println("Slack client stopped")
	os.Exit(0)
}

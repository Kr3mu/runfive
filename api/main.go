package main

import (
	"flag"
	"log"

	"github.com/Kr3mu/runfive/internal/api"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
)

var appConfig = fiber.Config{
	ServerHeader: "RunFive API",
	AppName:      "RunFive API",
}

var listenConfig = fiber.ListenConfig{
	DisableStartupMessage: true,
}

var humaConfig = huma.DefaultConfig("RunFive API", "0.01")

var listenPort = flag.String("port", "5000", "Starting ")

func main() {
	app := api.New(appConfig, humaConfig)

	log.Printf("Serving backend on: %s", *listenPort)
	log.Fatal(app.Listen(":"+*listenPort, listenConfig))
}

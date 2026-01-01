package main

import (
	"log"
	"paradigm-reboot-prober-go/internal/router"
)

// @title           Paradigm: Reboot Prober API
// @version         2
// @description     This is the API documentation for the Paradigm: Reboot Prober service.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      https://api.prp.icel.site
// @BasePath  /api/v2
func main() {
	r := router.SetupRouter()

	port := ":8080"
	log.Printf("Server starting on %s", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

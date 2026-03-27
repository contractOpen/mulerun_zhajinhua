package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"zhajinhua/game"
	"zhajinhua/handler"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "zhajinhua.db"
	}
	game.InitDB(dbPath)

	// Set app mode from environment
	appMode := strings.ToLower(os.Getenv("APP_MODE"))
	switch appMode {
	case "pe":
		handler.Mode = handler.ModePE
		log.Println("Running in PE mode (wallet required)")
	default:
		handler.Mode = handler.ModeTE
		log.Println("Running in TE mode (test, no wallet)")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Static files - serve from mode-specific dir if exists, fallback to ./static
	staticDir := fmt.Sprintf("./static-%s", handler.Mode)
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		staticDir = "./static"
	}
	fs := http.FileServer(http.Dir(staticDir))
	http.Handle("/", fs)

	http.HandleFunc("/ws", handler.HandleWebSocket)
	http.HandleFunc("/health", handler.HandleHealth)

	// Admin panel routes
	handler.RegisterAdminRoutes()
	log.Printf("Admin panel at /admin (password: ADMIN_PASSWORD env var)")

	// API endpoint to check mode
	http.HandleFunc("/api/mode", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"mode":"%s"}`, handler.Mode)
	})

	addr := fmt.Sprintf(":%s", port)
	log.Printf("炸金花服务器启动于 %s (mode: %s, static: %s)", addr, handler.Mode, staticDir)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}

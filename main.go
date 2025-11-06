package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/alibaba0010/postgres-api/logger"
	"github.com/alibaba0010/postgres-api/api/database"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)


func main(){
	err := godotenv.Load()
	if err != nil {
		// log.Println("No .env file found, using system environment variables...")
		logger.Log.Warn("No .env file found", zap.Error(err))
	}
	port := os.Getenv("PORT")
	if port == "" {
		port ="3001"
		fmt.Println("Add port to env file")
	}
	logger.InitLogger()
	// defer sync to flush logs on program exit
	defer logger.Sync()

	database.ConnectDB()
	defer database.CloseDB()

	route := mux.NewRouter()
	route.Use(logger.Logger)
	route.HandleFunc("/getUser", getUserHandler).Methods("GET")
	route.HandleFunc("/getBook", GetBookHandler).Methods("GET")
	route.HandleFunc("/", httpHandler).Methods("GET")
	logger.Log.Info("🚀 Server starting", zap.String("address", ":"+port))
	if  err:= http.ListenAndServe(":"+port, route); err != nil {
		log.Fatal(err)
	}
}

// http.HandleFunc("/getUser", getUserHandler)
// 	http.HandleFunc("/getBook", GetBookHandler)
// 	http.HandleFunc("/", httpHandler)
// if  err:= http.ListenAndServe(":"+port, nil); err != nil {

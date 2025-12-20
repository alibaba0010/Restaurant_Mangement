package main

import (
	"log"
	"net/http"

	_ "github.com/alibaba0010/postgres-api/docs" // swag doc
	"github.com/alibaba0010/postgres-api/internal/auth/routes"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
	"go.uber.org/zap"
)


func main(){

	logger.InitLogger()
	// defer sync to flush logs on program exit
	defer logger.Sync()

	database.ConnectDB()
	defer database.CloseDB()

	database.ConnectRedis()
	defer database.CloseRedis()

	port := config.LoadConfig().Port
	route := routes.ApiRouter()

	
	logger.Log.Info("🚀 Server starting", zap.String("url", "http://localhost:"+port+"/swagger/index.html"))
	if  err:= http.ListenAndServe(":"+port, route); err != nil {
		log.Fatal(err)
	}
}

// http.HandleFunc("/getUser", getUserHandler)
// 	http.HandleFunc("/getBook", GetBookHandler)
// 	http.HandleFunc("/", httpHandler)
// if  err:= http.ListenAndServe(":"+port, nil); err != nil {

package main

import (
	"fmt"
	"log"
	"net/http"

	"go.uber.org/zap"

	_ "github.com/alibaba0010/postgres-api/docs" // swagger docs
	"github.com/alibaba0010/postgres-api/internal/auth/routes"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/middlewares"
	"github.com/alibaba0010/postgres-api/internal/config"
	"github.com/alibaba0010/postgres-api/internal/database"
)


func main(){
	fmt.Print("\033[H\033[2J")

	logger.InitLogger()
	// defer sync to flush logs on program exit
	defer logger.Sync()

	database.ConnectDB()
	defer database.CloseDB()

	database.ConnectRedis()
	defer database.CloseRedis()
	
	cfg := config.LoadConfig()

	port := cfg.Port
	frontendUrl := cfg.FRONTEND_URL
	route := routes.ApiRouter()
	
	logger.Log.Info("🚀 Server Listening on ", zap.String("url", "http://localhost:"+port+"/swagger/index.html"))
	
	// Apply CORS middleware globally
	handler := middlewares.CORS(frontendUrl)(route)
	
	if  err:= http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"flag"

	log "github.com/golang/glog"
	_ "github.com/nchc-ai/rfstack/docs"
	"github.com/nchc-ai/rfstack/rfserver"
	"github.com/nchc-ai/rfstack/stackclient"
)

// @title rfstack API
// @version 0.2
// @description AI Train VM API.

// @host localhost:8088
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

func main() {

	configPath := flag.String("conf", "", "The file path to a config file")
	flag.Parse()

	//setDefault()
	config, err := rfserver.ReadConfig(*configPath)
	if err != nil {
		log.Fatalf("Unable to read configure file: %s", err.Error())
	}

	db, err := stackclient.NewDBClient(config)
	if err != nil {
		log.Fatalf("Faild to Connect to Database: %s", err.Error())
	}

	provider, err := stackclient.NewStackClient(config.GetString("stackvar.tenantid"), config.GetString("rfserver.provider.endpoint"), config.GetString("rfserver.provider.username"), config.GetString("rfserver.provider.password"))
	if err != nil {
		log.Fatalf("Faild to Connect to OpenStack: %s", err.Error())
		return
	}

	server := rfserver.NewRFServer(config, provider, db)
	if server == nil {
		log.Fatalf("Create Restful server fail, Stop!!")
		return
	}

	log.Info("Start Restful Stack Server")
	err = server.RunServer()
	if err != nil {
		log.Fatalf("start api server error: %s", err.Error())
	}
}

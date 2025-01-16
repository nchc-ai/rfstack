package rfserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
	"github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"

	"strings"

	github_provider "github.com/nchc-ai/github-oauth-provider/pkg/provider"
	go_provider "github.com/nchc-ai/go-oauth-provider/pkg/provider"
	google_provider "github.com/nchc-ai/google-oauth-provider/pkg/provider"
	provider_config "github.com/nchc-ai/oauth-provider/pkg/config"
	"github.com/nchc-ai/oauth-provider/pkg/provider"
	"github.com/nchc-ai/rfstack/util"
)

type RFServer struct {
	config        *viper.Viper
	router        *gin.Engine
	client        *gophercloud.ProviderClient
	mysqldb       *gorm.DB
	providerProxy provider.Provider
}

func NewRFServer(config *viper.Viper, client *gophercloud.ProviderClient, db *gorm.DB) *RFServer {
	providerConfigstr := config.GetStringMapString("api-server.provider")
	var vconf provider_config.ProviderConfig

	// map[string]string -> json
	jsonStr, err := json.Marshal(providerConfigstr)
	if err != nil {
		log.Fatalf(fmt.Sprintf(":%s", err.Error()))
		return nil
	}

	// json -> struct
	err = json.Unmarshal([]byte(jsonStr), &vconf)
	if err != nil {
		log.Fatalf(fmt.Sprintf(":%s", err.Error()))
		return nil
	}

	var providerProxy provider.Provider
	switch oauthProvider := vconf.Type; oauthProvider {
	case go_provider.GO_OAUTH:
		providerProxy = go_provider.NewGoAuthProvider(vconf)
	case github_provider.GITHUB_OAUTH:
		providerProxy = github_provider.NewGitAuthProvider(vconf)
	case google_provider.GOOGLE_OAUTH:
		providerProxy = google_provider.NewGoogleAuthProvider(vconf)
	default:
		log.Warning(fmt.Sprintf("%s is a not supported provider type", oauthProvider))
	}

	return &RFServer{
		config:        config,
		router:        gin.Default(),
		client:        client,
		mysqldb:       db,
		providerProxy: providerProxy,
	}
}

func (server *RFServer) RunServer() error {

	server.router.Use(server.CORSHeaderMiddleware())
	server.AddRoute(server.router)
	err := server.router.Run(":" + strconv.Itoa(server.config.GetInt("rfserver.port")))
	if err != nil {
		log.Fatalf("Start api server error: %s", err.Error())
		return err
	}
	return nil
}

func (server *RFServer) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Error("Authorization header is missing")
			util.RespondWithError(c, http.StatusUnauthorized, "Authorization header is missing")
			return
		}

		bearerToken := strings.Split(authHeader, " ")

		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			log.Errorf("Authorization header is not Bearer Token format or token is missing: %s", authHeader)
			util.RespondWithError(c, http.StatusUnauthorized, "Authorization header is not Bearer Token format or token is missing")
			return
		}

		var validated bool
		var err error

		token := bearerToken[1]
		validated, err = server.providerProxy.Validate(token)

		if err != nil && err.Error() == "Access token expired" {
			log.Error("Access token expired")
			util.RespondWithError(c, http.StatusForbidden, "Access token expired")
			return
		}

		if err != nil {
			log.Errorf("verify token fail: %s", err.Error())
			util.RespondWithError(c, http.StatusInternalServerError, "verify token fail: %s", err.Error())
			return
		}

		if !validated {
			util.RespondWithError(c, http.StatusForbidden, "Invalid API token")
			return
		}

		provider := fmt.Sprintf("%s:%s", server.providerProxy.Type(), server.providerProxy.Name())
		c.Set("Provider", provider)
		c.Next()

	}
}

func (server *RFServer) CORSHeaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		c.Next()
	}
}

func (server *RFServer) handleOption(c *gin.Context) {
	//	setup headers
	c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Access-Control-Allow-Origin, Access-Control-Allow-Credentials")
	c.Status(http.StatusOK)
}

func (server *RFServer) AddRoute(router *gin.Engine) {

	course := router.Group("/v1").Group("/course")
	{
		course.GET("/list", server.ListAllCourse)
		course.POST("/search", server.SearchCourse)
		course.GET("/level/:level", server.ListLevelCourse)
		course.OPTIONS("/create", server.handleOption)
		course.OPTIONS("/delete/:id", server.handleOption)
		course.OPTIONS("/list", server.handleOption)
		course.OPTIONS("/get/:id", server.handleOption)
		course.OPTIONS("/update", server.handleOption)
		course.OPTIONS("/search", server.handleOption)
		course.OPTIONS("/level/:level", server.handleOption)
	}

	courseAuth := router.Group("/v1").Group("/course").Use(server.AuthMiddleware())
	{
		courseAuth.POST("/create", server.AddCourse)
		courseAuth.POST("/list", server.ListUserCourse)
		courseAuth.DELETE("/delete/:id", server.DeleteCourse)
		courseAuth.GET("/get/:id", server.GetCourse)
		courseAuth.PUT("/update", server.UpdateCourse)
	}

	job := router.Group("/v1").Group("/job")
	{
		job.OPTIONS("/list", server.handleOption)
		job.OPTIONS("/launch", server.handleOption)
		job.OPTIONS("/delete/:id", server.handleOption)
	}

	jobAuth := router.Group("/v1").Group("/job").Use(server.AuthMiddleware())
	{
		jobAuth.POST("/list", server.ListJob)
		jobAuth.POST("/launch", server.LaunchJob)
		jobAuth.DELETE("/delete/:id", server.DeleteJob)
	}

	classroom := router.Group("/v1").Group("/classroom")
	{
		classroom.OPTIONS("/delete/:id", server.handleOption)
	}

	classroomAuth := router.Group("/v1").Group("/classroom").Use(server.AuthMiddleware())
	{
		classroomAuth.DELETE("/delete/:id", server.DeleteClassroomJobs)
	}

	vm := router.Group("/v1").Group("/vm")
	{
		vm.OPTIONS("/snapshot", server.handleOption)
		vm.OPTIONS("stop/:id", server.handleOption)
		vm.OPTIONS("start/:id", server.handleOption)
	}

	vmAuth := router.Group("/v1").Group("/vm").Use(server.AuthMiddleware())
	{
		vmAuth.POST("/snapshot", server.SnapshotVM)
		vmAuth.GET("stop/:id", server.StopVM)
		vmAuth.GET("start/:id", server.StartVM)
	}

	image := router.Group("/v1").Group("/image")
	{
		image.OPTIONS("/list", server.handleOption)
	}

	imageAuth := router.Group("/v1").Group("/image").Use(server.AuthMiddleware())
	{
		imageAuth.GET("/list", server.ListImage)
	}

	flavor := router.Group("/v1").Group("/flavor")
	{
		flavor.OPTIONS("/list", server.handleOption)
	}

	flavorAuth := router.Group("/v1").Group("/flavor").Use(server.AuthMiddleware())
	{
		flavorAuth.GET("/list", server.ListFlavor)
	}

	key := router.Group("/v1").Group("/key")
	{
		key.OPTIONS("/list", server.handleOption)
	}

	keyAuth := router.Group("/v1").Group("/key").Use(server.AuthMiddleware())
	{
		keyAuth.GET("/list", server.ListKey)
	}

	/*
	   share := router.Group("/v1").Group("/share").Use(server.AuthMiddleware())
	   {
	       share.GET("/list", server.ListShare)
	   }
	*/

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

}

func ReadConfig(fileConfig string) (*viper.Viper, error) {
	viper := viper.New()
	viper.SetConfigType("json")
	if fileConfig == "" {
		viper.SetConfigName("server-config")
		viper.AddConfigPath("conf")
	} else {
		viper.SetConfigFile(fileConfig)
		//glog.Fatalf("Load Config")
	}
	// overwrite by file
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Unable to read configure file: %s", err.Error())
		return nil, err
	}

	return viper, nil
}

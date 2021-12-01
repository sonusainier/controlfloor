package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/gin-gonic/gin"
	_ "github.com/nanoscopic/controlfloor/docs"
	cfauth "github.com/nanoscopic/controlfloor_auth"
	adminauth "github.com/nanoscopic/controlfloor_auth_admin"
	uc "github.com/nanoscopic/uclop/mod"
	swagFiles "github.com/swaggo/files"
	swag "github.com/swaggo/gin-swagger"
)

func main() {
	uclop := uc.NewUclop()
	uclop.AddCmd("run", "Run ControlFloor", runMain, nil)
	uclop.AddCmd("devs", "List registered devices", runListDevs, nil)
	uclop.AddCmd("prov", "List providers", runListProv, nil)
	uclop.AddCmd("conf", "Dump configuration", runDumpConf, nil)
	uclop.Run()
}

func runDumpConf(*uc.Cmd) {
	conf := NewConfig("config.json", "default.json")
	fmt.Printf("%s\n", conf)
}

func runListDevs(*uc.Cmd) {
	openDbConnection()

	var devices []DbDevice
	err := gDb.Find(&devices)
	if err != nil {
		panic(err)
	}

	for _, device := range devices {
		fmt.Printf("Name: %s\nUdid: %s\nProvider Id: %d\n\n",
			device.Name, device.Udid, device.ProviderId)
	}
}

func runListProv(*uc.Cmd) {
	openDbConnection()

	var provs []DbProvider
	err := gDb.Find(&provs)
	if err != nil {
		panic(err)
	}

	for _, prov := range provs {
		fmt.Printf("Username: %s\nProvider Id: %d\n\n",
			prov.Username, prov.Id)
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST,HEAD,PATCH, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func runMain(*uc.Cmd) {
	conf := NewConfig("config.json", "default.json")

	openDbConnection()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(CORSMiddleware())

	initTemplates(r, conf)
	r.Static("/assets", "./assets")
	sessionManager := NewSessionManager(r)

	devTracker := NewDevTracker(conf)

	var authHandler cfauth.AuthHandler
	if conf.auth == "mod" {
		authHandler = cfauth.NewAuthHandler(conf.root, sessionManager)
	}

	var adminHandler adminauth.AuthHandler
	if conf.adminAuth == "mod" {
		adminHandler = adminauth.NewAuthHandler(conf.root.Get("adminAuth"), sessionManager)
	}

	uh := NewUserHandler(authHandler, r, devTracker, sessionManager, conf)
	uAuth := uh.registerUserRoutes()

	ah := NewAdminHandler(adminHandler, r, devTracker, sessionManager, conf)
	aAuth := ah.registerAdminRoutes()

	ph := NewProviderHandler(r, devTracker, sessionManager)
	pAuth := ph.registerProviderRoutes()

	dh := NewDevHandler(pAuth, uAuth, aAuth, devTracker, sessionManager, conf)
	dh.registerDeviceRoutes()

	th := NewTestHandler(r, sessionManager)
	th.registerTestRoutes()

	swagFunc := swag.WrapHandler(swagFiles.Handler)
	r.GET("/swagger/*any", func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/swagger/" {
			c.Redirect(302, "/swagger/index.html")
			return
		}
		swagFunc(c)
	})

	var err error
	protocol := "http"
	if conf.https {
		protocol = "https"
		if conf.crt == "server.crt" && !fileExists("server.crt") {
			gen_cert()
		}
		err = http.ListenAndServeTLS(conf.listen, conf.crt, conf.key, r)
	} else {
		err = http.ListenAndServe(conf.listen, r)
	}
	fmt.Printf("%s ListenAndServe Error %s\n", protocol, err)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func gen_cert() {
	out, err := exec.Command("/usr/bin/perl", "gencert.pl").Output()
	if err != nil {
		fmt.Printf("Error from cert gen: %s\n", err)
		return
	}
	fmt.Println(out)
}

func censorUuid(uuid string) string {
	return "***" + uuid[len(uuid)-4:]
}

// @title ControlFloor API
// @version 1.0
// @description ControlFloor Server API

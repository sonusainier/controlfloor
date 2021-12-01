package main

import (
    "fmt"
    "net/http"
    "encoding/json"
    "github.com/gin-gonic/gin"
    adminauth "github.com/nanoscopic/controlfloor_auth_admin"
    //log "github.com/sirupsen/logrus"
)

type AdminHandler struct {
    authHandler    adminauth.AuthHandler
    r              *gin.Engine
    devTracker     *DevTracker
    sessionManager *cfSessionManager
    config         *Config
}

func NewAdminHandler(
    authHandler    adminauth.AuthHandler,
    r              *gin.Engine,
    devTracker     *DevTracker,
    sessionManager *cfSessionManager,
    config         *Config,
) *AdminHandler {
    return &AdminHandler{
        authHandler,
        r,
        devTracker,
        sessionManager,
        config,
    }
}

func (self *AdminHandler) registerAdminRoutes() (*gin.RouterGroup) {
    r := self.r
    
    fmt.Println("Registering admin routes")
    r.GET("/admin/login", self.showAdminLogin )
    r.GET("/admin/logout", self.handleAdminLogout )
    r.POST("/admin/login", self.handleAdminLogin )
    aAuth := r.Group("/admin")
    aAuth.Use( self.NeedAdminAuth( self.authHandler ) )
    aAuth.GET("/", self.showAdminRoot )
    return aAuth
}

func (self *AdminHandler) NeedAdminAuth( authHandler adminauth.AuthHandler ) gin.HandlerFunc {
    return func( c *gin.Context ) {
        sCtx := self.sessionManager.GetSession( c )
        
        loginI := self.sessionManager.session.Get( sCtx, "admin" )
        
        if loginI == nil {
            if authHandler != nil {
                authHandler.UserAuth( c )
                return
            }
            
            c.Redirect( 302, "/admin/login" )
            c.Abort()
            fmt.Println("admin user fail")
            return
        } else {
            fmt.Println("admin user ok")
        }
        
        c.Next()
    }
}

// @Summary Home - Admin
// @Router /admin/ [GET]
func (self *AdminHandler) showAdminRoot( c *gin.Context ) {
    devices, err := getDevices()
    if err != nil { panic( err ) }
    
    output := ""
    for _, device := range devices {
        output = output + fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td><a href="/devInfo?udid=%s">%s</a></td>
                <td>%d</td>
                <td>%s</td>
                <td>%d</td><td>%d</td><td>%d</td><td>%d</td>
            </tr>`,
            device.Name,
            device.Udid, device.Udid,
            device.ProviderId,
            device.JsonInfo,
            device.Width,
            device.Height,
            device.ClickWidth,
            device.ClickHeight,
        )
        // also Width, Height, ClickWidth, and ClickHeight
    }
    
    rs, _ := getReservations()
    if rs == nil {
        rs = make( map[string]DbReservation )
    }
    
    sCtx := self.sessionManager.GetSession( c )
    user := self.sessionManager.session.Get( sCtx, "admin" ).(string)
    
    jsont := ""
    for _, device := range devices {
        udid := device.Udid
        
        provId := self.devTracker.getDevProvId( udid )
        if provId != 0 {
            r, hasR := rs[ udid ]
            if hasR && r.User != user {
                device.Ready = "In Use"
            } else {
                device.Ready = "Yes"
            }
        } else {
            device.Ready = "No"
        }
        
        t, _ := json.Marshal( device )
              
        jsont += string(t) + ","
    }
    if jsont != "" {
        jsont = jsont[:len( jsont )-1]
    }
    
    c.HTML( http.StatusOK, "adminRoot", gin.H{
        "devices":      output,
        "devices_json": jsont,
        "deviceVideo":  self.config.text.deviceVideo,
    } )
}

func (self *AdminHandler) showAdminLogin( rCtx *gin.Context ) {
    rCtx.HTML( http.StatusOK, "adminLogin", gin.H{} )
}

// @Description Admin - Logout
// @Router /adminLogout [POST]
func (self *AdminHandler) handleAdminLogout( c *gin.Context ) {
    s := self.sessionManager.GetSession( c )
    
    self.sessionManager.session.Remove( s, "admin" )
    self.sessionManager.WriteSession( c )
    
    c.Redirect( 302, "/adminLogin" )
}

// @Description Admin - Login
// @Router /adminlogin [POST]
// @Param user formData string true "Username"
// @Param pass formData string true "Password"
func (self *AdminHandler) handleAdminLogin( c *gin.Context ) {
    if self.authHandler != nil {
        success := self.authHandler.UserLogin( c )
        if success {
            c.Redirect( 302, "/admin/" )
        } else {
            fmt.Printf("admin login failed\n")
            self.showAdminLogin( c )
        }
        return
    }
    
    s := self.sessionManager.GetSession( c )
    
    user := c.PostForm("user")
    pass := c.PostForm("pass")
    
    if user == "ok" && pass == "ok" {
        fmt.Printf("admin login ok\n")
        
        self.sessionManager.session.Put( s, "admin", "test" )
        self.sessionManager.WriteSession( c )
        
        c.Redirect( 302, "/admin/" )
        return
    } else {
        fmt.Printf("admin login failed\n")
    }
    
    self.showAdminLogin( c )
}
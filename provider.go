package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
	uj "github.com/nanoscopic/ujsonin/v2/mod"
	log "github.com/sirupsen/logrus"
)

type ProviderOb struct {
	User string
	Id   int64
}

type ProviderHandler struct {
	r              *gin.Engine
	devTracker     *DevTracker
	sessionManager *cfSessionManager
}

func NewProviderHandler(
	r *gin.Engine,
	devTracker *DevTracker,
	sessionManager *cfSessionManager,
) *ProviderHandler {
	return &ProviderHandler{
		r,
		devTracker,
		sessionManager,
	}
}

func (self *ProviderHandler) registerProviderRoutes() *gin.RouterGroup {
	r := self.r

	fmt.Println("Registering provider routes")
	r.POST("/provider/register", self.handleRegister)
	r.GET("/provider/login", self.showProviderLogin)
	r.GET("/provider/logout", self.handleProviderLogout)
	r.POST("/provider/login", self.handleProviderLogin)

	pAuth := r.Group("/provider")
	pAuth.Use(self.NeedProviderAuth())
	pAuth.GET("/", self.showProviderRoot)
	pAuth.GET("/ws", func(c *gin.Context) {
		self.handleProviderWS(c)
	})
	pAuth.GET("/imgStream", func(c *gin.Context) {
		self.handleImgProvider(c)
	})

	return pAuth
}

func (self *ProviderHandler) NeedProviderAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		sCtx := self.sessionManager.GetSession(c)

		_, ok := self.sessionManager.session.Get(sCtx, "provider").(ProviderOb)
		// provider not used

		if !ok {
			c.Redirect(302, "/provider/login")
			c.Abort()
			fmt.Println("provider fail")
			return
		} else {
			//fmt.Printf("provider user=%s\n", provider.User )
		}

		c.Next()
	}
}

var wsupgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	CMKick  = iota
	CMPing  = iota
	CMFrame = iota
)

type ClientMsg struct {
	msgType int
	msg     string
}

type FrameMsg struct {
	msg       int
	frame     []byte
	frameType int
}

// @Description Provider - Image Stream Websocket
// @Router /provider/imgStream [GET]
func (self *ProviderHandler) handleImgProvider(c *gin.Context) {
	//s := getSession( c )

	//provider := session.Get( s, "provider" ).(ProviderOb)

	udid, uok := c.GetQuery("udid")
	if !uok {
		c.HTML(http.StatusOK, "error", gin.H{
			"text": "no uuid set",
		})
		return
	}
	log.WithFields(log.Fields{
		"type": "provider_video_start",
		"udid": censorUuid(udid),
	}).Info("Provider -> Server video connected")

	//dev := getDevice( udid )

	provId := self.devTracker.getDevProvId(udid)
	provConn := self.devTracker.getProvConn(provId)

	writer := c.Writer
	req := c.Request
	conn, err := wsupgrader.Upgrade(writer, req, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}

	vidConn := self.devTracker.getVidStreamOutput(udid)
	outSocket := vidConn.socket
	clientOffset := vidConn.offset

	msgChan := make(chan ClientMsg)
	self.devTracker.addClient(udid, msgChan)

	frameChan := make(chan FrameMsg, 20)

	// Consume incoming frames as fast as possible only ever holding onto the latest frame
	go func() {
		ingestDone := false
		for {
			if ingestDone == true {
				break
			}
			t, data, err := conn.ReadMessage()
			//fmt.Printf("Got frame\n")
			if err != nil {
				conn = nil
				if frameChan != nil {
					frameChan <- FrameMsg{
						msg:       CMKick,
						frame:     []byte{},
						frameType: 0,
					}
				}
				fmt.Printf("Frame receive error: %s\n", err)
				break
			}
			if frameChan != nil {
				frameChan <- FrameMsg{
					msg:       CMFrame,
					frame:     data,
					frameType: t,
				}
			}

			select {
			case msg := <-msgChan:
				outSocket.WriteMessage(ws.TextMessage, []byte(msg.msg))
				if msg.msgType == CMKick {
					fmt.Printf("Got kick from client; ending ingest\n")
					if frameChan != nil {
						frameChan <- FrameMsg{
							msg:       CMKick,
							frame:     []byte{},
							frameType: 0,
						}
					}
					ingestDone = true
					break
				}
			default:
			}
		}
	}()

	var frameSleep int32
	frameSleep = 0

	go func() {
		for {
			_, data, err := outSocket.ReadMessage()
			if err != nil {
				break
			}
			root, _ := uj.Parse(data)
			bpsNode := root.Get("bps")
			if bpsNode != nil {
				avgFrameStr := root.Get("avgFrame").String()
				avgFrame, _ := strconv.ParseInt(avgFrameStr, 10, 64)

				bpsStr := bpsNode.String()
				bps, _ := strconv.ParseInt(bpsStr, 10, 64)
				if bps != 10000000 {
					fpsMax := (float64(bps) / float64(avgFrame)) * 0.75
					delayMs := float32(1000) / float32(fpsMax)
					//fmt.Printf("fpsMax: %d ; delayMs: %d\n", fpsMax, delayMs )
					frameSleep = int32(delayMs)
				}
			}
		}
	}()

	abort := false

	go func() {
		for {
			if abort {
				return
			}
			if frameChan != nil {
				frameChan <- FrameMsg{
					msg:       CMPing,
					frame:     []byte{},
					frameType: 0,
				}
			} else {
				break
			}
			time.Sleep(time.Second)
		}
	}()

	// Whenever a frame is ready send the latest frame
	for {
		if abort {
			fmt.Printf("Frame sender got CMKick. Aborting\n")
			break
		}

		var frame FrameMsg
		gotFrame := false
		emptied := false
		for {
			select {
			case msg := <-frameChan:
				if msg.msg == CMKick {
					frameChan = nil
					abort = true
				} else if msg.msg == CMPing {
					awriter, err := outSocket.NextWriter(ws.TextMessage)
					if err == nil {
						_, err = awriter.Write([]byte("ping"))
						if err == nil {
							err = awriter.Close()
						}
					}
					if err != nil {
						frameChan <- FrameMsg{
							msg:       CMKick,
							frame:     []byte{},
							frameType: 0,
						}
					}
					continue
				} else {
					gotFrame = true
					frame = msg
				}
				break
			default:
				emptied = true
			}
			if emptied {
				break
			}
		}

		if !gotFrame {
			time.Sleep(time.Millisecond * time.Duration(20))
			continue
		}

		if abort {
			fmt.Printf("Frame sender got CMKick. Aborting\n")
			break
		}
		toSend := frame.frame
		t := frame.frameType

		//fmt.Printf("Sending frame to client\n")

		if t == ws.TextMessage {
			// Just send the message and continue
			outSocket.WriteMessage(ws.TextMessage, frame.frame)
			continue
		}

		var timeBeforeSend int64
		var writer io.WriteCloser
		var err error
		if t != ws.TextMessage {
			writer, err = outSocket.NextWriter(ws.TextMessage)
			if err == nil {
				nowMilli := time.Now().UnixMilli() + clientOffset
				nowBytes := []byte(strconv.FormatInt(nowMilli, 10))
				writer.Write(nowBytes)
				writer.Close()

				writer, err = outSocket.NextWriter(t)
				if err == nil {
					timeBeforeSend = time.Now().UnixMilli()
					nowMilli = timeBeforeSend + clientOffset

					nowBytes = []byte(fmt.Sprintf("%*d", 100, nowMilli))
					toSend = append(toSend, nowBytes...)
				}
			}
		}
		if err != nil {
			fmt.Printf("Error creating outSocket writer: %s\n", err)
			outSocket = nil
			provConn.stopImgStream(udid)
			break
		}

		_, err = writer.Write(toSend)
		if err == nil {
			err = writer.Close()
		}
		if err != nil {
			fmt.Printf("Error writing frame: %s\n", err)
			outSocket = nil
			provConn.stopImgStream(udid)
			frameChan <- FrameMsg{
				msg:       CMKick,
				frame:     []byte{},
				frameType: 0,
			}
		}

		if frameSleep == 0 {
			continue
		}

		timeAfterSend := time.Now().UnixMilli()

		timeSending := int32(timeAfterSend - timeBeforeSend)

		if timeSending > frameSleep {
			continue
		}

		milliToSleep := frameSleep - timeSending

		time.Sleep(time.Millisecond * time.Duration(milliToSleep))
	}

	log.WithFields(log.Fields{
		"type": "provider_video_end",
		"udid": censorUuid(udid),
	}).Info("Provider -> Server video disconnected")

	self.devTracker.delVidStreamOutput(udid, vidConn.rid)
	self.devTracker.deleteClient(udid)

	if conn != nil {
		conn.Close()
	}
	if outSocket != nil {
		outSocket.Close()
	}
}

// @Description Provider - Websocket
// @Router /provider/ws [GET]
func (self *ProviderHandler) handleProviderWS(c *gin.Context) {
	s := self.sessionManager.GetSession(c)

	provider := self.sessionManager.session.Get(s, "provider").(ProviderOb)

	writer := c.Writer
	req := c.Request
	conn, err := wsupgrader.Upgrade(writer, req, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}

	provChan := make(chan ProvBase)
	provConn := NewProviderConnection(provChan)
	self.devTracker.setProvConn(provider.Id, provConn)
	reqTracker := provConn.reqTracker
	reqTracker.conn = conn

	amDone := false

	fmt.Printf("Provider Connection Established - Provider:%s\n", provider.User)

	go func() {
		for {
			time.Sleep(time.Second * 5)
			provConn.doPing(func(root uj.JNode, raw []byte) {
				text := root.Get("text").String()
				if text != "pong" {
					amDone = true
				}
			})

			if amDone {
				if provChan != nil {
					provChan <- nil
				}
				break
			}
		}
	}()

	go func() {
		for {
			t, msg, err := conn.ReadMessage()
			if err != nil {
				amDone = true
			} else {
				jsonroot := reqTracker.processResp(t, msg)
				if jsonroot != nil {
					// This is not a response; is a request from provider

				}
			}

			if amDone {
				if provChan != nil {
					provChan <- nil
				}
				break
			}
		}
	}()

	for {
		ev := <-provChan
		if ev == nil {
			provChan = nil
			break
		}

		err, reqText := reqTracker.sendReq(ev)
		if err != nil {
			fmt.Printf("Failed to send request to provider\n")
			fmt.Printf("  Request data:%s\n", reqText)
			fmt.Printf("  Error:%s\n", err)
			provConn.provChan = nil
			amDone = true
			break
		}
	}

	self.devTracker.clearProvConn(provider.Id)
	fmt.Printf("Provider Connection Lost - Provider:%s\n", provider.User)
}

func randHex() string {
	c := 16
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

type SProviderRegistration struct {
	Success  bool   `json:"Success"  example:"true"`
	Password string `json:"Password" example:"huefw3fw3"`
	Existed  bool   `json:"Existed"  example:"false"`
}

// @Description Provider - Register
// @Router /provider/register [POST]
// @Param regPass formData string true "Registration password"
// @Param username formData string true "Provider username"
// @Produce json
// @Success 200 {object} SProviderRegistration
func (self *ProviderHandler) handleRegister(c *gin.Context) {
	pass := c.PostForm("regPass")

	conf := getConf()
	if pass != conf.RegPass {
		var jsonf struct {
			Success bool
		}
		jsonf.Success = false
		c.JSON(http.StatusOK, jsonf)
		return
	}

	username := c.PostForm("username")

	var json struct {
		Success  bool
		Password string
		Existed  bool
	}
	json.Success = true
	pPass := randHex()
	json.Password = pPass
	existed := addProvider(username, pPass)
	json.Existed = existed

	c.JSON(http.StatusOK, json)
}

// @Description Provider - Login
// @Router /provider/login [POST]
// @Param user query string true "Username"
// @Param pass query string true "Password"
func (self *ProviderHandler) handleProviderLogin(c *gin.Context) {
	s := self.sessionManager.GetSession(c)

	user := c.PostForm("user")
	pass := c.PostForm("pass")
	fmt.Printf("Provider login user=%s pass=%s\n", user, pass)

	// ensure the user is legit
	provider := getProvider(user)
	if provider == nil {
		fmt.Printf("provider login failed 1\n")
		c.Redirect(302, "/provider/?fail=1")
		return
	}

	if pass == provider.Password {
		//fmt.Printf("provider login ok\n")

		self.sessionManager.session.Put(s, "provider", &ProviderOb{
			User: user,
			Id:   provider.Id,
		})
		self.sessionManager.WriteSession(c)

		c.Redirect(302, "/provider/")
		return
	} else {
		fmt.Printf("provider login failed [submit]%s != [db]%s\n", pass, provider.Password)
		c.Redirect(302, "/provider/?fail=2")
		return
	}

	self.showProviderLogin(c)
}

// @Description Provider - Logout
// @Router /provider/logout [GET]
func (self *ProviderHandler) handleProviderLogout(c *gin.Context) {
	s := self.sessionManager.GetSession(c)

	self.sessionManager.session.Remove(s, "provider")
	self.sessionManager.WriteSession(c)

	c.Redirect(302, "/")
}

func (self *ProviderHandler) showProviderLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "providerLogin", gin.H{})
}

func (self *ProviderHandler) showProviderRoot(c *gin.Context) {
	c.HTML(http.StatusOK, "providerRoot", gin.H{})
}

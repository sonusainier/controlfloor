package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
	uj "github.com/nanoscopic/ujsonin/v2/mod"
)

type ProvSafariTestMsg struct {
	Id   int16  `json:"id"`
	Type string `json:"type"`
	Udid string `json:"udid"`
	Url  string `json:"url"`
}

type ProvSafariUrl struct {
	udid  string
	url   string
	onRes func(uj.JNode, []byte)
}

func (self *ProvSafariUrl) resHandler() func(data uj.JNode, rawData []byte) {
	return self.onRes
}
func (self *ProvSafariUrl) needsResponse() bool { return true }
func (self *ProvSafariUrl) asText(id int16) string {
	msg := ProvSafariTestMsg{
		Id:   id,
		Type: "launchsafariurl",
		Udid: self.udid,
		Url:  self.url,
	}
	res, _ := json.Marshal(msg)
	return string(res)
}

type ProvBrowserCleanUpMsg struct {
	Id   int16  `json:"id"`
	Type string `json:"type"`
	Udid string `json:"udid"`
	Bid  string `json:"bid"`
}

type ProvBrowserCleanup struct {
	udid  string
	bid   string
	onRes func(uj.JNode, []byte)
}

func (self *ProvBrowserCleanup) resHandler() func(data uj.JNode, rawData []byte) {
	return self.onRes
}
func (self *ProvBrowserCleanup) needsResponse() bool { return true }
func (self *ProvBrowserCleanup) asText(id int16) string {
	msg := ProvBrowserCleanUpMsg{
		Id:   id,
		Type: "cleanbrowser",
		Udid: self.udid,
		Bid:  self.bid,
	}
	res, _ := json.Marshal(msg)
	return string(res)
}

type SDeviceRefresh struct {
	Udid    string `json:"udid"        example:"00008100-001338811EE10033"`
	Refresh string `json:"refresh"          example:"unknown or x.x.x.x"`
}

type SDeviceRestart struct {
	Udid    string `json:"udid"        example:"00008100-001338811EE10033"`
	Restart string `json:"restart"          example:"unknown or x.x.x.x"`
}

func (self *DevHandler) handleDeviceRefresh(c *gin.Context) {
	udid, uok := c.GetQuery("udid")
	if !uok {
		c.JSON(http.StatusOK, SDeviceInfoFail{
			Success: false,
			Err:     "Must pass udid",
		})
		return
	}

	dev := getDevice(udid)
	if dev == nil {
		c.JSON(http.StatusOK, SDeviceInfoFail{
			Success: false,
			Err:     "No device with that udid",
		})
		return
	}

	//

	provId := self.devTracker.getDevProvId(udid)
	pc := self.devTracker.getProvConn(provId)

	done := make(chan bool)

	refresh := "unknown"

	pc.doRefresh(udid, func(_ uj.JNode, json []byte) {
		root, _ := uj.Parse(json)

		refresh = root.Get("refresh").String()

		done <- true
	})

	<-done

	//

	c.JSON(http.StatusOK, SDeviceRefresh{
		Udid:    udid,
		Refresh: refresh,
	})
}

func (self *DevHandler) handleDeviceRestart(c *gin.Context) {
	udid, uok := c.GetQuery("udid")
	if !uok {
		c.JSON(http.StatusOK, SDeviceInfoFail{
			Success: false,
			Err:     "Must pass udid",
		})
		return
	}

	dev := getDevice(udid)
	if dev == nil {
		c.JSON(http.StatusOK, SDeviceInfoFail{
			Success: false,
			Err:     "No device with that udid",
		})
		return
	}

	provId := self.devTracker.getDevProvId(udid)
	pc := self.devTracker.getProvConn(provId)

	done := make(chan bool)

	restart := "false"
	pc.doRestart(udid, func(_ uj.JNode, json []byte) {
		root, _ := uj.Parse(json)

		restart = root.Get("restart").String()

		done <- true
	})

	<-done

	//

	c.JSON(http.StatusOK, SDeviceRestart{
		Udid:    udid,
		Restart: restart,
	})
}

// @Summary Device - Launch url in safari app
// @Router /device/launchsafariurl [POST]
// @Param udid formData string true "Device UDID"
// @Param bid formData string true "[bundle id]"
func (self *DevHandler) handleSafariUrl(c *gin.Context) {
	url := c.PostForm("url")
	pc, udid := self.getPc(c)

	done := make(chan bool)

	pc.doOpenSafariUrl(udid, url, func(uj.JNode, []byte) {
		done <- true
	})

	<-done

	c.HTML(http.StatusOK, "error", gin.H{
		"text": "ok",
	})
}

// @Summary Device - Launch url in safari app
// @Router /device/cleansafari [POST]
// @Param udid formData string true "Device UDID"
// @Param bid formData string true "[bundle id]"
func (self *DevHandler) handleBrowserCleanup(c *gin.Context) {
	bid := c.PostForm("bid")
	pc, udid := self.getPc(c)

	done := make(chan bool)

	pc.doBrowserCleanup(udid, bid, func(uj.JNode, []byte) {
		done <- true
	})

	<-done

	c.HTML(http.StatusOK, "error", gin.H{
		"text": "ok",
	})
}

func (self *DevHandler) handleStreamingRestart(c *gin.Context) {
	udid, uok := c.GetQuery("udid")
	if !uok {
		c.JSON(http.StatusOK, SDeviceInfoFail{
			Success: false,
			Err:     "Must pass udid",
		})
		return
	}

	dev := getDevice(udid)
	if dev == nil {
		c.JSON(http.StatusOK, SDeviceInfoFail{
			Success: false,
			Err:     "No device with that udid",
		})
		return
	}

	provId := self.devTracker.getDevProvId(udid)
	pc := self.devTracker.getProvConn(provId)

	done := make(chan bool)

	restart := "false"
	pc.doRestart(udid, func(_ uj.JNode, json []byte) {
		root, _ := uj.Parse(json)

		restart = root.Get("restart").String()

		done <- true
	})

	<-done

	//

	c.JSON(http.StatusOK, SDeviceRestart{
		Udid:    udid,
		Restart: restart,
	})
}

var upGrader = ws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (self *DevHandler) echo(c *gin.Context) {

	conn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	for {

		msgType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}

		var msg map[string]interface{}
		json.Unmarshal(message, &msg)

		if msg["event"] == "click" {

			udid := msg["udid"].(string)
			x := int(msg["x"].(float64))
			y := int(msg["y"].(float64))

			self.handleDevClickWS(udid, x, y)

		} else if msg["event"] == "swipe" {

			udid := msg["udid"].(string)
			x1 := int(msg["x1"].(float64))
			y1 := int(msg["y1"].(float64))
			x2 := int(msg["x2"].(float64))
			y2 := int(msg["y2"].(float64))
			delay := msg["delay"].(float64)

			self.handleDevSwipeWS(udid, x1, y1, x2, y2, delay)

		} else if msg["event"] == "keys" {

			udid := msg["udid"].(string)
			keys := msg["keys"].(string)
			curid := int(msg["curid"].(float64))
			prevkeys := msg["prevkeys"].(string)

			self.handleKeysWS(udid, keys, curid, prevkeys)

		}

		msg_received, _ := json.Marshal(msg)

		if err = conn.WriteMessage(msgType, msg_received); err != nil {
			log.Println("write:", err)
			return
		}
	}
}

// @Summary Device - Click coordinate
// @Router /device/click [POST]
// @Param udid formData string true "Device UDID"
// @Param x formData int true "x"
// @Param y formData int true "y"
func (self *DevHandler) handleDevClickWS(udid string, x int, y int) {

	pc, udid := self.getPcWS(udid)
	if pc == nil {
		fmt.Println("click : not ok")
		return
	}

	done := make(chan bool)

	pc.doClick(udid, x, y, func(uj.JNode, []byte) {
		done <- true
	})

	<-done

}

// @Summary Device - Swipe
// @Router /device/swipe [POST]
// @Param udid formData string true "Device UDID"
// @Param x1 formData int true "x1"
// @Param y1 formData int true "y1"
// @Param x2 formData int true "x2"
// @Param y2 formData int true "y2"
// @Param delay formData number true "Time of swipe"
func (self *DevHandler) handleDevSwipeWS(udid string, x1 int, y1 int, x2 int, y2 int, delay float64) {

	pc, udid := self.getPcWS(udid)
	if pc == nil {
		fmt.Println("swipe : not ok")
		return
	}

	done := make(chan bool)

	pc.doSwipe(udid, x1, y1, x2, y2, delay, func(uj.JNode, []byte) {
		done <- true
	})

	<-done

}

// @Summary Device - Simulate keystrokes
// @Router /device/keys [POST]
// @Param udid formData string true "Device UDID"
// @Param curid formData int true "Incrementing unique ID"
// @Param keys formData string true "Keys"
// @Param prevkeys formData string true "Previous keys"
func (self *DevHandler) handleKeysWS(udid string, keys string, curid int, prevkeys string) {

	done := make(chan bool)

	pc, udid := self.getPcWS(udid)
	if pc == nil {
		fmt.Println("keys : not ok")
		return
	}

	pc.doKeys(udid, keys, curid, prevkeys, func(uj.JNode, []byte) {
		done <- true
	})

	<-done

}

func (self *DevHandler) getPcWS(udid string) (*ProviderConnection, string) {
	// udid := c.PostForm("udid")
	provId := self.devTracker.getDevProvId(udid)
	provConn := self.devTracker.getProvConn(provId)
	if provConn == nil {
		fmt.Printf("Could not get provider for udid:%s\n", udid)
	}
	return provConn, udid
}

package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	uj "github.com/nanoscopic/ujsonin/v2/mod"
)

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
	return fmt.Sprintf("{id:%d,type:\"launchsafariurl\",udid:\"%s\",url:\"%s\"}\n", id, self.udid, self.url)
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

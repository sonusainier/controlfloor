package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	uj "github.com/nanoscopic/ujsonin/v2/mod"
)

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

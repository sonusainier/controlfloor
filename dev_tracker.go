package main

import (
	"sync"

	ws "github.com/gorilla/websocket"
)

type VidConn struct {
	socket *ws.Conn
	//stopChan chan bool
	onDone func()
	offset int64
	rid    string
}

type NoticeConn struct {
	socket *ws.Conn
}

type DevStatus struct {
	wda   bool
	cfa   bool
	video bool
}

type DevInfo struct {
	orientation string
}

type DevTracker struct {
	provConns   map[int64]*ProviderConnection
	devToProv   map[string]int64
	DevStatus   map[string]*DevStatus
	vidConns    map[string]*VidConn
	DevInfo     map[string]*DevInfo
	noticeConns map[string]*NoticeConn
	clients     map[string]chan ClientMsg
	lock        *sync.Mutex
	config      *Config
}

func NewDevTracker(config *Config) *DevTracker {
	self := &DevTracker{
		provConns:   make(map[int64]*ProviderConnection),
		devToProv:   make(map[string]int64),
		lock:        &sync.Mutex{},
		vidConns:    make(map[string]*VidConn),
		noticeConns: make(map[string]*NoticeConn),
		DevStatus:   make(map[string]*DevStatus),
		DevInfo:     make(map[string]*DevInfo),
		clients:     make(map[string]chan ClientMsg),
		config:      config,
	}

	return self
}

func (self *DevTracker) delVidStreamOutput(udid string, rid string) {
	self.lock.Lock()
	curConn, exists := self.vidConns[udid]
	if exists {
		if curConn.rid != rid {
			return
		}
		onDone := curConn.onDone
		delete(self.vidConns, udid)
		self.lock.Unlock()
		onDone()
		return
	}
	//delete( self.vidConns, udid )
	self.lock.Unlock()
}

func (self *DevTracker) setVidStreamOutput(udid string, vidConn *VidConn) {
	self.lock.Lock()
	curConn, exists := self.vidConns[udid]
	if exists {
		onDone := curConn.onDone
		self.vidConns[udid] = vidConn
		self.lock.Unlock()
		onDone()
		return
	}
	self.vidConns[udid] = vidConn
	self.lock.Unlock()
}

func (self *DevTracker) getVidStreamOutput(udid string) *VidConn {
	return self.vidConns[udid]
}

func (self *DevTracker) delNoticeOutput(udid string, rid string) {
	self.lock.Lock()
	_, exists := self.noticeConns[udid]
	if exists {
		delete(self.noticeConns, udid)
		// TODO: Something to cleanup
	}
	self.lock.Unlock()
}

func (self *DevTracker) setNoticeOutput(udid string, noticeConn *NoticeConn) {
	self.lock.Lock()
	_, exists := self.noticeConns[udid]
	if exists {
		// TODO: Something to cleanup the old one

		self.noticeConns[udid] = noticeConn
		self.lock.Unlock()
		return
	}
	self.noticeConns[udid] = noticeConn
	self.lock.Unlock()
}

func (self *DevTracker) getNoticeOutput(udid string) *NoticeConn {
	return self.noticeConns[udid]
}

func (self *DevTracker) setDevProv(udid string, provId int64) {
	self.lock.Lock()
	self.devToProv[udid] = provId
	self.DevStatus[udid] = &DevStatus{}
	self.lock.Unlock()
}

func (self *DevTracker) clearDevProv(udid string) {
	self.lock.Lock()
	delete(self.devToProv, udid)
	delete(self.DevStatus, udid)
	self.lock.Unlock()
}

func (self *DevTracker) addClient(udid string, msgChan chan ClientMsg) {
	self.lock.Lock()
	self.clients[udid] = msgChan
	self.lock.Unlock()
}

func (self *DevTracker) deleteClient(udid string) {
	self.lock.Lock()
	delete(self.clients, udid)
	self.lock.Unlock()
}

func (self *DevTracker) msgClient(udid string, msg ClientMsg) {
	msgChan, chanOk := self.clients[udid]
	if !chanOk {
		return
	}
	msgChan <- msg
}

func (self *DevTracker) setDevStatus(udid string, service string, status bool) {
	stat, statOk := self.DevStatus[udid]
	if !statOk {
		return
	}
	if service == "wda" {
		stat.wda = status
		return
	}
	if service == "cfa" {
		stat.cfa = status
		return
	}
	if service == "video" {
		stat.video = status
		return
	}
}

func (self *DevTracker) getDevInfo(udid string) *DevInfo {
	devInfo, exists := self.DevInfo[udid]
	if !exists {
		devInfo = &DevInfo{}
		self.DevInfo[udid] = devInfo
	}
	return devInfo
}

func (self *DevTracker) getDevStatus(udid string) *DevStatus {
	devStatus, devOk := self.DevStatus[udid]
	if devOk {
		return devStatus
	} else {
		return nil
	}
}

func (self *DevTracker) getDevProvId(udid string) int64 {
	provId, provOk := self.devToProv[udid]
	if provOk {
		return provId
	} else {
		return 0
	}
}

func (self *DevTracker) setProvConn(provId int64, provConn *ProviderConnection) {
	self.lock.Lock()
	self.provConns[provId] = provConn
	self.lock.Unlock()
}

func (self *DevTracker) getProvConn(provId int64) *ProviderConnection {
	return self.provConns[provId]
}

func (self *DevTracker) clearProvConn(provId int64) {
	self.lock.Lock()
	delete(self.provConns, provId)
	self.lock.Unlock()
}

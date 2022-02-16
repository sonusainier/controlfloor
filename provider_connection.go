package main

import (
	"fmt"
)

type ProviderConnection struct {
	//    provChan chan ProvBase
	provChan   chan *CFRequest
	reqTracker *ReqTracker
}

func NewProviderConnection(provChan chan *CFRequest) *ProviderConnection {
	self := &ProviderConnection{
		provChan:   provChan,
		reqTracker: NewReqTracker(),
	}

	return self
}

func (self *ProviderConnection) initWebrtc(udid string, offer string, onDone func(CFResponse)) {
	//    action := &ProvInitWebrtc{
	//        udid: udid,
	//        offer: offer,
	//        onRes: onDone,
	//    }
	//    action := &CFRequest{
	//        CFDeviceID: udid,
	//        Offer: offer,
	//        onRes: onDone,
	//    }
	action := NewCFRequest("initWebRTC", WebRTCRequest{Offer: offer})
	action.onRes = onDone
	if self == nil || self.provChan == nil {
		//errorChannelGone( action );
		return
	}
	self.provChan <- action
}

func errorChannelGone(message *CFRequest) {
	b, _ := message.JSONBytes()
	fmt.Printf("Failed to send message to provider:\n")
	fmt.Printf("  %s\n", string(b))
}

func (self *ProviderConnection) doShutdown(onDone func(CFResponse)) {
	msg := &CFRequest{
		Action: "shutdown",
		onRes:  onDone,
	}
	if self == nil || self.provChan == nil {
		//    errorChannelGone( msg );
		return
	}
	self.provChan <- msg
}

func (self *ProviderConnection) startImgStream(udid string) {
	if self == nil || self.provChan == nil {
		//errorChannelGone( &ProvStartStream{ udid: udid } );
		return
	}
	cfrequest := NewCFRequest(CFStartVideoStream, nil)
	cfrequest.CFDeviceID = udid
	self.provChan <- cfrequest
}

func (self *ProviderConnection) stopImgStream(udid string) {
	if self == nil || self.provChan == nil {
		//errorChannelGone( &ProvStopStream{ udid: udid } );
		return
	}
	cfrequest := NewCFRequest(CFStopVideoStream, nil)
	cfrequest.CFDeviceID = udid
	self.provChan <- cfrequest
}

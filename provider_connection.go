package main

import (
    "fmt"
)

type ProviderConnection struct {
//    provChan chan ProvBase
    provChan chan *CFRequest
    reqTracker *ReqTracker
}

func NewProviderConnection( provChan chan *CFRequest ) (*ProviderConnection) {
    self := &ProviderConnection{
        provChan: provChan,
        reqTracker: NewReqTracker(),
    }
    
    return self
}

func (self *ProviderConnection) initWebrtc( udid string, offer string, onDone func( CFResponse ) ) {
//    action := &ProvInitWebrtc{
//        udid: udid,
//        offer: offer,
//        onRes: onDone,
//    }
    action := &CFRequest{
        CFDeviceID: udid,
        Offer: offer,
        onRes: onDone,
    }
    if self == nil || self.provChan == nil { 
        //errorChannelGone( action ); 
        return 
    }
    self.provChan <- action
}


func errorChannelGone( message *CFRequest ) {
    fmt.Printf("Failed to send message to provider:\n")
    fmt.Printf("  %s\n", message.asText(0) )
}


func (self *ProviderConnection) doShutdown( onDone func( CFResponse ) ) {
    msg := &CFRequest{
        Action:"shutdown",
        onRes: onDone,
    }
    if self == nil || self.provChan == nil { 
    //    errorChannelGone( msg ); 
        return 
    }
    self.provChan <- msg
}

func (self *ProviderConnection) startImgStream( udid string ) {
    if self == nil || self.provChan == nil { 
        //errorChannelGone( &ProvStartStream{ udid: udid } ); 
        return 
    }
    self.provChan <- &CFRequest{Action:"startVideoStream",CFDeviceID:udid} // &ProvStartStream{ udid: udid }
}

func (self *ProviderConnection) stopImgStream( udid string ) {
    if self == nil || self.provChan == nil { 
        //errorChannelGone( &ProvStopStream{ udid: udid } ); 
        return
    }
    self.provChan <- &CFRequest{Action:"stopVideoStream",CFDeviceID:udid} // &ProvStartStream{ udid: udid }
//    self.provChan <- &ProvStopStream{ udid: udid }
}








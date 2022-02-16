package main

import (
    "fmt"
    mrand "math/rand"
//    "strings"
    "sync"
    ws "github.com/gorilla/websocket"
//    uj "github.com/nanoscopic/ujsonin/v2/mod"
)

type ReqTracker struct {
    reqMap map[int16] *CFRequest //ProvBase
    lock *sync.Mutex
    conn *ws.Conn
    //sequenceNumber int32;
}

func NewReqTracker() (*ReqTracker) {
    self := &ReqTracker{
        reqMap: make( map[int16] *CFRequest ),
//        reqMap: make( map[int16] ProvBase ),
        lock: &sync.Mutex{},
        //sequenceNumber : 1,
    }
    
    return self
}

//func (self *ReqTracker) sendReq( req ProvBase ) (error,string) {
func (self *ReqTracker) sendReq( req *CFRequest ) (error,string) {
    var reqText string
//    req.CFServerRequestID = self.sequenceNumber
//    self.sequenceNumber++
    if req.RequiresResponse || req.onRes != nil{ // Even if a response is not required, a response may still come (errors).  TODO: this would potentially a giant memory leak
        var id int16
        maxi := ^uint16(0) / 2
        self.lock.Lock()
        for {
            id = int16( mrand.Int31n( int32(maxi-2) ) ) + 1
            _, exists := self.reqMap[ id ]
            if !exists { break }
        }
        
        self.reqMap[ id ] = req
        self.lock.Unlock()
//        reqText = req.asText( id )
        req.CFServerRequestID = int(id)
      }
//    } else {
//        reqText = req.asText( 0 )
//    }
    
//    if !strings.Contains( reqText, "ping" ) {
        bytes,_ := req.JSONBytes()
        fmt.Printf("sending %s\n", string(bytes) )
//    }
    // send the request
    err := self.conn.WriteMessage( ws.TextMessage, bytes) //[]byte(reqText) )
    if err != nil {
        return err,reqText
    }
    return err, ""
}

func (self *ReqTracker) processResp( msgType int, reqText []byte ) *CFResponse {
    cfresponse ,err := NewCFResponseFromJSON(reqText)
//    err := json.Unmarshal(reqText, &cfresponse)
    
    if err!=nil{
        fmt.Printf("Could not decode response from provider: %v",err)
        return nil
    }
    fmt.Println("Received response: %s",string(reqText))
    id := cfresponse.CFServerRequestID
    
    if id == 0 {
        return cfresponse
    }
    
    req := self.reqMap[ int16(id) ]
    if req == nil{
        fmt.Println("Error: received message with invalid or duplicated id %d",id)
        return nil
    }
    //TODO:
    resHandler := req.onRes //resHandler()
    if resHandler != nil {
        resHandler( *cfresponse )
    }
    
    self.lock.Lock()
    delete( self.reqMap, int16(id) )
    self.lock.Unlock()
    // deserialize the reqText to get the id
    // fetch the original request from the reqMap
    // respond to the original request if needed
    return nil
}

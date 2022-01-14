package main

import (
    "fmt"
    mrand "math/rand"
    "strings"
    "sync"
    ws "github.com/gorilla/websocket"
    uj "github.com/nanoscopic/ujsonin/v2/mod"
)

type ReqTracker struct {
    reqMap map[int16] ProvBase
    lock *sync.Mutex
    conn *ws.Conn
}

func NewReqTracker() (*ReqTracker) {
    self := &ReqTracker{
        reqMap: make( map[int16] ProvBase ),
        lock: &sync.Mutex{},
    }
    
    return self
}

func (self *ReqTracker) sendReq( req ProvBase ) (error,string) {
    var reqText string
    if req.needsResponse() {
        var id int16
        maxi := ^uint16(0) / 2
        for {
            id = int16( mrand.Int31n( int32(maxi-2) ) ) + 1
            _, exists := self.reqMap[ id ]
            if !exists { break }
        }
        
        self.lock.Lock()
        self.reqMap[ id ] = req
        self.lock.Unlock()
        reqText = req.asText( id )
    } else {
        reqText = req.asText( 0 )
    }
    
    if !strings.Contains( reqText, "ping" ) {
        fmt.Printf("sending %s\n", reqText )
    }
    // send the request
    err := self.conn.WriteMessage( ws.TextMessage, []byte(reqText) )
    if err != nil {
        return err,reqText
    }
    return err, ""
}

func (self *ReqTracker) processResp( msgType int, reqText []byte ) uj.JNode {
    if !strings.Contains( string(reqText), "pong" ) {
        fmt.Printf( "received %s\n", string(reqText) )
    }
    
    if len( reqText ) == 0 {
        return nil
    }
    c1 := string( []byte{ reqText[0] } )
    if c1 != "{" {
        return nil
    }
    last1 := string( []byte{ reqText[ len( reqText ) - 1 ] } )
    last2 := string( []byte{ reqText[ len( reqText ) - 2 ] } )
    if last1 != "}" && last2 != "}" {
        fmt.Printf("response not json; last1=%s\n", last1)
        return nil
    }
    
    root, _, err := uj.ParseFull( reqText )
    if err != nil {
        fmt.Printf("Could not parse response as json\n")
        return nil
    }
    
    id := root.Get("id").Int()
    
    if id == 0 {
        return root
    }
    
    req := self.reqMap[ int16(id) ]
    resHandler := req.resHandler()
    if resHandler != nil {
        resHandler( root, reqText )
    }
    
    self.lock.Lock()
    delete( self.reqMap, int16(id) )
    self.lock.Unlock()
    // deserialize the reqText to get the id
    // fetch the original request from the reqMap
    // respond to the original request if needed
    return nil
}
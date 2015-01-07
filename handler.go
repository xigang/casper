package casper

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/cascades-fbp/cascades/runtime"
	. "github.com/gogap/base_component"
)

const (
	REQ_X_API   = "X-API"
	REQ_TIMEOUT = time.Duration(15) * time.Second
)

type Response struct {
	Code    uint64      `json:"code"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

func handle(p *App) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		p.logger.Debug("http Handler:", r.Method, r.RequestURI)

		apiName := r.Header.Get(REQ_X_API)
		if apiName == "" {
			http.NotFound(w, r)
			return
		}
		port := p.GetApi(apiName)
		if port == nil {
			http.NotFound(w, r)
			return
		}

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			p.logger.Errorln("request body err:", p.Name, err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Read request body error"))
			return
		}
		p.logger.Debug("req:", apiName, string(reqBody))

		componentMsg, _ := NewComponentMessage()
		componentMsg.Payload.Result = reqBody

		ch := p.AddRequest(componentMsg.ID)
		defer p.DelRequest(componentMsg.ID)
		defer close(ch)

		// Send Component message
		msgBytes, err := componentMsg.Serialize()
		if err != nil {
			p.logger.Errorln("Service Internal Error")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Service Internal Error"))
			return
		}
		p.logger.Infoln("ToNextComponent:", port.OutPort[0].url, string(msgBytes))
		port.OutPort[0].socket.SendMessage(runtime.NewPacket(msgBytes))

		// Wait for response from IN port
		p.logger.Debug("Waiting for response from a channel port (from INPUT port)")
		var load *Payload
		select {
		case load = <-ch:
			break
		case <-time.Tick(REQ_TIMEOUT):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Couldn't process request in a given time"))
			return
		}

		p.logger.Infoln("Data arrived. Responding to HTTP response...")
		objResp := Response{
			Code:    load.Code,
			Message: load.Message,
			Result:  load.Result}

		bResp, _ := json.Marshal(objResp)
		w.Write(bResp)
	}
}
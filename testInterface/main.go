package main

import (
	"github.com/op/go-logging"
	"github.com/peersafe/gohfc"
	"encoding/json"
)

var logger = logging.MustGetLogger("testmodel")

func main() {
	err := gohfc.InitSDK("./client.yaml")
	if err != nil {
		logger.Error(err)
		return
	}
	ch, err := gohfc.GetHandler().ListenEventFullBlock("mychannel")
	if err != nil {
		logger.Error(err)
		return
	}

	for {
		select {
		case b := <- ch:
			bytes, _ := json.Marshal(b)
			logger.Debugf("---------%s\n",bytes)
		}
	}
}

package main

import (
	"github.com/op/go-logging"
	"github.com/peersafe/gohfc"
	"fmt"
	"flag"
	"encoding/json"
)

var (
	logger = logging.MustGetLogger("testmodel")
	funcName = flag.String("function", "", "invoke,query,listen")
)

func main() {
	flag.Parse()
	err := gohfc.InitSDK("./client.yaml")
	if err != nil {
		logger.Error(err)
		return
	}

	switch *funcName {
	case "invoke":
		res, err := gohfc.GetHandler().Invoke([]string{"invoke", "a", "b", "1"})
		if err != nil {
			logger.Error(err)
			return
		}
		logger.Debugf("----invoke--TxID--%s\n", res.TxID)
	case "query":
		resVal, err := gohfc.GetHandler().Query([]string{"query", "a"})
		if err != nil || len(resVal) == 0 {
			logger.Error(err)
			return
		}
		if resVal[0].Error != nil {
			logger.Error(resVal[0].Error)
			return
		}
		if resVal[0].Response.Response.GetStatus() != 200 {
			logger.Error(fmt.Errorf(resVal[0].Response.Response.GetMessage()))
			return
		}
		logger.Debugf("----query--result--%s\n",resVal[0].Response.Response.GetPayload())
	case "listen":
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
	default:
		flag.PrintDefaults()
	}
	return
}

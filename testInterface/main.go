package main

import (
	"github.com/op/go-logging"
	"github.com/peersafe/gohfc"
	"fmt"
	"flag"
	"encoding/json"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/core/scc/cbcc/define"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/golang/protobuf/proto"
	mspprotos "github.com/hyperledger/fabric/protos/msp"
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
	case "configupdate":
		channel := "mychannel"
		msppath := "/opt/gopath/src/github.com/peersafe/worktool/crypto-config/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp"

		mspConfig, err := msp.GetLocalMspConfig(msppath, nil, "Org2MSP")
		if err != nil {
			fmt.Println("GetLocalMspConfig Error = ", err)
			return
		}

		config := &mspprotos.FabricMSPConfig{}
		if err := proto.Unmarshal(mspConfig.Config, config); err != nil {
			return
		}

		// CreatFlow == AddMember
		addMember := &define.ContentAddMember{}
		addMember.Channel = channel
		member := &define.Member{
			MspId:     "Org2MSP",
			Name:      "测试成员1",
			Role:      define.OPERATION,
			Status:    0,
			MSPConfig: mspConfig,
		}

		member.AnchorInfo = append(member.AnchorInfo, &peer.AnchorPeer{Host: "peer0.org2.example.com", Port: 7250})
		addMember.Members = append(addMember.Members, member)

		// add member
		//flow := &define.Flow{
		//	Type:define.ADD_MEMBERS,
		//	Content:addMember,
		//}

		// update anchor
		//flow := &define.Flow{
		//	Type:define.MODIFY_ANCHOR_PEERS,
		//	Content:&define.ContentModifyAnchor{
		//		MspId:"Org1MSP",
		//		Channel:channel,
		//		AnchorInfo:member.AnchorInfo,
		//	},
		//}

		// create channel
		flow := &define.Flow{
			Type:define.CREAT_CHANNEL,
			Content:&define.Channel{
				Channel:"channeltest",
				MemberList:[]string{"Org1MSP"},
				AnchorInfo:[][]*peer.AnchorPeer{member.AnchorInfo},
			},
		}

		data, _ := json.Marshal(flow)

		resVal, err := gohfc.GetHandler().Query([]string{"GetConfigUpdateEnv", string(data)})
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
		fmt.Println("build tx success")
		err = gohfc.GetHandler().ConfigUpdate(resVal[0].Response.Response.GetPayload(), "channeltest")
		if err != nil {
			fmt.Println("config update error, err : ", err)
			return
		}
		fmt.Println("config update success")
	case "querytest":
		resVal, err := gohfc.GetHandler().Query([]string{"getval", "test"})
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
		ch, err := gohfc.GetHandler().ListenEventFullBlock()
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

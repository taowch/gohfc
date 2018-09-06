package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/peersafe/gohfc"
	"strconv"
)

func main() {
	conf := gohfc.PeerConfig{
		Host:     "peer1.org1.hbhafifc.com:7051",
		Insecure: false,
		TlsPath:  "/opt/gopath/src/github.com/peersafe/bcap/build/install/networklist/hbhafifc/crypto-config/peerOrganizations/org1.hbhafifc/peers/peer1.org1.hbhafifc.com/tls/server.crt",
		OrgName:  "org1.hbhafifc",
	}
	cryptoFamily := "sm2"
	pubkey := "/opt/gopath/src/github.com/peersafe/bcap/build/install/networklist/hbhafifc/crypto-config/peerOrganizations/org1.hbhafifc/peers/peer1.org1.hbhafifc.com/msp/signcerts"
	prikey := "/opt/gopath/src/github.com/peersafe/bcap/build/install/networklist/hbhafifc/crypto-config/peerOrganizations/org1.hbhafifc/peers/peer1.org1.hbhafifc.com/msp/keystore"
	mspId := "org1MSPhbhafifc"
	chainId := "channelpooxlrsv"
	blockNum := uint64(0)

	if _, err := GetBlockByNumber(conf, cryptoFamily, pubkey, prikey, mspId, chainId, blockNum); err != nil {
		fmt.Println("GetBlockByNumber err : ", err)
		return
	}
	fmt.Println("success")
}

func GetBlockByNumber(conf gohfc.PeerConfig, cryptoFamily, pubkey, prikey, mspId, chainId string, blockNum uint64) (*common.Block, error) {
	strBlockNum := strconv.FormatUint(blockNum, 10)
	args := []string{"GetBlockByNumber", chainId, strBlockNum}
	resps, err := gohfc.QueryQscc(conf, cryptoFamily, pubkey, prikey, mspId, chainId, args)
	if err != nil {
		return nil, fmt.Errorf("can not get installed chaincodes :%s", err.Error())
	}
	if resps.Error != nil {
		return nil, resps.Error
	}
	data := resps.Response.Response.Payload
	var block = new(common.Block)
	err = proto.Unmarshal(data, block)
	if err != nil {
		return nil, fmt.Errorf("GetBlockByNumber Unmarshal from payload failed: %s", err.Error())
	}

	return block, nil
}

package gohfc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/op/go-logging"
	"github.com/peersafe/gohfc/parseBlock"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

//sdk handler
type sdkHandler struct {
	client   *FabricClient
	identity *Identity
}

var (
	logger          = logging.MustGetLogger("gohfc")
	handler         sdkHandler
	orgPeerMap      = make(map[string][]string)
	orderNames      []string
	eventName       string
	orRulePeerNames []string
)

func InitSDK(configPath string) error {
	// initialize Fabric client
	var err error
	handler.client, err = NewFabricClient(configPath)
	if err != nil {
		return err
	}
	mspPath := handler.client.Channel.MspConfigPath
	if mspPath == "" {
		return fmt.Errorf("yaml mspPath is empty")
	}
	findCert := func(path string) string {
		list, err := ioutil.ReadDir(path)
		if err != nil {
			return ""
		}
		var file os.FileInfo
		for _, item := range list {
			if !item.IsDir() {
				if file == nil {
					file = item
				} else if item.ModTime().After(file.ModTime()) {
					file = item
				}
			}
		}
		return filepath.Join(path, file.Name())
	}
	prikey := findCert(filepath.Join(mspPath, "keystore"))
	pubkey := findCert(filepath.Join(mspPath, "signcerts"))
	if prikey == "" || pubkey == "" {
		return fmt.Errorf("prikey or cert is no such file")
	}
	handler.identity, err = LoadCertFromFile(pubkey, prikey)
	if err != nil {
		return err
	}
	handler.identity.MspId = handler.client.Channel.LocalMspId

	if err := setLogLevel(); err != nil {
		return fmt.Errorf("setLogLevel err: %s\n", err.Error())
	}

	if err := parsePolicy(); err != nil {
		return fmt.Errorf("parsePolicy err: %s\n", err.Error())
	}
	return err
}

// GetHandler get sdk handler
func GetHandler() *sdkHandler {
	return &handler
}

// Invoke invoke cc
func (sdk *sdkHandler) Invoke(args []string) (*InvokeResponse, error) {
	peerNames := getSendPeerName()
	orderName := getSendOrderName()
	if len(peerNames) == 0 || orderName == "" {
		return nil, fmt.Errorf("config peer order is err")
	}
	chaincode, err := getChainCodeObj(args)
	if err != nil {
		return nil, err
	}
	return sdk.client.Invoke(*sdk.identity, *chaincode, peerNames, orderName)
}

// Query query cc
func (sdk *sdkHandler) Query(args []string) ([]*QueryResponse, error) {
	peerNames := getSendPeerName()
	if len(peerNames) == 0 {
		return nil, fmt.Errorf("config peer order is err")
	}
	chaincode, err := getChainCodeObj(args)
	if err != nil {
		return nil, err
	}

	return sdk.client.Query(*sdk.identity, *chaincode, []string{peerNames[0]})
}

// Query query qscc
func (sdk *sdkHandler) QueryByQscc(args []string) ([]*QueryResponse, error) {
	peerNames := getSendPeerName()
	if len(peerNames) == 0 {
		return nil, fmt.Errorf("config peer order is err")
	}
	channelid := handler.client.Channel.ChannelId
	mspId := handler.client.Channel.LocalMspId
	if channelid == "" || mspId == "" {
		return nil, fmt.Errorf("channelid or ccname or mspid is empty")
	}

	chaincode := ChainCode{
		ChannelId: channelid,
		Type:      ChaincodeSpec_GOLANG,
		Name:      QSCC,
		Args:      args,
	}

	return sdk.client.Query(*sdk.identity, chaincode, []string{peerNames[0]})
}

func (sdk *sdkHandler) GetBlockByNumber(blockNum uint64) (*common.Block, error) {
	strBlockNum := strconv.FormatUint(blockNum, 10)
	args := []string{"GetBlockByNumber", sdk.client.Channel.ChannelId, strBlockNum}
	logger.Debugf("GetBlockByNumber chainId %s num %s", sdk.client.Channel.ChannelId, strBlockNum)
	resps, err := sdk.QueryByQscc(args)
	if err != nil {
		return nil, fmt.Errorf("can not get installed chaincodes :%s", err.Error())
	} else if len(resps) == 0 {
		return nil, fmt.Errorf("GetBlockByNumber empty response from peer")
	}
	if resps[0].Error != nil {
		return nil, resps[0].Error
	}
	data := resps[0].Response.Response.Payload
	var block = new(common.Block)
	err = proto.Unmarshal(data, block)
	if err != nil {
		return nil, fmt.Errorf("GetBlockByNumber Unmarshal from payload failed: %s", err.Error())
	}

	return block, nil
}

func (sdk *sdkHandler) GetBlockHeight() (uint64, error) {
	args := []string{"GetChainInfo", sdk.client.Channel.ChannelId}
	//logger.Debugf("GetBlockHeight chainId %s", sdk.client.Channel.ChannelId)
	resps, err := sdk.QueryByQscc(args)
	if err != nil {
		return 0, err
	} else if len(resps) == 0 {
		return 0, fmt.Errorf("GetChainInfo is empty respons from peer qscc")
	}

	if resps[0].Error != nil {
		return 0, resps[0].Error
	}

	data := resps[0].Response.Response.Payload
	var chainInfo = new(common.BlockchainInfo)
	err = proto.Unmarshal(data, chainInfo)
	if err != nil {
		return 0, fmt.Errorf("GetChainInfo unmarshal from payload failed: %s", err.Error())
	}
	return chainInfo.Height, nil
}

func (sdk *sdkHandler) ListenEventFullBlock() (chan parseBlock.Block, error) {
	channelId := sdk.client.Channel.ChannelId
	if channelId == "" {
		return nil, fmt.Errorf("ListenEventFullBlock channelId is empty ")
	}
	ch := make(chan parseBlock.Block)
	ctx, cancel := context.WithCancel(context.Background())
	err := sdk.client.ListenForFullBlock(ctx, *sdk.identity, eventName, channelId, ch)
	if err != nil {
		cancel()
		return nil, err
	}
	//
	//for d := range ch {
	//	fmt.Println(d)
	//}
	return ch, nil
}

func (sdk *sdkHandler) ListenEventFilterBlock() (chan EventBlockResponse, error) {
	channelId := sdk.client.Channel.ChannelId
	if channelId == "" {
		return nil, fmt.Errorf("ListenEventFilterBlock  channelId is empty ")
	}
	ch := make(chan EventBlockResponse)
	ctx, cancel := context.WithCancel(context.Background())
	err := sdk.client.ListenForFilteredBlock(ctx, *sdk.identity, eventName, channelId, ch)
	if err != nil {
		cancel()
		return nil, err
	}
	//
	//for d := range ch {
	//	fmt.Println(d)
	//}
	return ch, nil
}

// Listen v 1.0.4 -- port ==> 7053
func (sdk *sdkHandler) Listen(peerName string) (chan parseBlock.Block, error) {
	channelId := sdk.client.Channel.ChannelId
	if channelId == "" {
		return nil, fmt.Errorf("Listen  channelId is empty ")
	}
	mspId := sdk.client.Channel.LocalMspId
	if mspId == "" {
		return nil, fmt.Errorf("Listen  mspId is empty ")
	}
	ch := make(chan parseBlock.Block)
	ctx, cancel := context.WithCancel(context.Background())
	err := sdk.client.Listen(ctx, sdk.identity, peerName, channelId, mspId, ch)
	if err != nil {
		cancel()
		return nil, err
	}
	return ch, nil
}

//解析区块
func (sdk *sdkHandler) ParseCommonBlock(block *common.Block) (*parseBlock.Block, error) {
	blockObj := parseBlock.ParseBlock(block, 0)
	return &blockObj, nil
}

type KeyValue struct {
	Key   string `json:"key"`   //存储数据的key
	Value string `json:"value"` //存储数据的value
}

func SetArgsTxid(txid string, args *[]string) {
	if len(*args) == 2 && (*args)[0] == "SaveData" {
		var invokeRequest KeyValue
		if err := json.Unmarshal([]byte((*args)[1]), &invokeRequest); err != nil {
			logger.Debugf("SetArgsTxid umarshal invokeRequest failed")
			return
		}
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(invokeRequest.Value), &msg); err != nil {
			logger.Debugf("SetArgsTxid umarshal message failed")
			return
		}
		invokeRequest.Key = txid
		msg["fabricTxId"] = txid
		v, _ := json.Marshal(msg)
		invokeRequest.Value = string(v)
		tempData, _ := json.Marshal(invokeRequest)
		//logger.Debugf("SetArgsTxid msg is %s", tempData)
		(*args)[1] = string(tempData)
	}
}

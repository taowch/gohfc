package gohfc

import (
	"context"
	"fmt"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path/filepath"
)

var sdklogger = logging.MustGetLogger("gohfc")

//sdk handler
type sdkHandler struct {
	client   *FabricClient
	identity *Identity
}

var handler sdkHandler

func InitSDK(configPath string) error {
	// initialize Fabric client
	var err error
	handler.client, err = NewFabricClient(configPath)
	if err != nil {
		sdklogger.Debugf("Error loading file %s err: %v", configPath, err)
		return err
	}
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	if err != nil {
		sdklogger.Debugf("Read file failed:", err.Error())
		return err
	}
	mspPath := viper.GetString("other.mspConfigPath")
	if mspPath == "" {
		return fmt.Errorf("yaml mspPath is empty")
	}
	findCert := func(path string) string {
		list, err := ioutil.ReadDir(path)
		if err != nil {
			sdklogger.Debug(err.Error())
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
	sdklogger.Debugf("privateKey : %s", prikey)
	sdklogger.Debugf("publicKey : %s", pubkey)
	handler.identity, err = LoadCertFromFile(pubkey, prikey)
	if err != nil {
		sdklogger.Debugf("load cert from file failed:", err.Error())
		return err
	}

	return err
}

// GetHandler get sdk handler
func GetHandler() *sdkHandler {
	return &handler
}

// Invoke invoke cc
func (sdk *sdkHandler) Invoke(args []string, peers []string, ordername string) (*InvokeResponse, error) {
	chaincode, err := getChainCodeObj(args)
	if err != nil {
		return nil, err
	}
	return sdk.client.Invoke(*sdk.identity, *chaincode, peers, ordername)
}

// Query query cc
func (sdk *sdkHandler) Query(args []string, peers []string) ([]*QueryResponse, error) {
	chaincode, err := getChainCodeObj(args)
	if err != nil {
		return nil, err
	}

	return sdk.client.Query(*sdk.identity, *chaincode, peers)
}

// Query query qscc
func (sdk *sdkHandler) QueryByQscc(args []string, peers []string) ([]*QueryResponse, error) {
	channelid := viper.GetString("other.channelId")
	mspId := viper.GetString("other.localMspId")
	if channelid == "" || mspId == "" {
		return nil, fmt.Errorf("channelid or ccname or mspid is empty")
	}

	chaincode := ChainCode{
		ChannelId: channelid,
		Type:      ChaincodeSpec_GOLANG,
		Name:      QSCC,
		Args:      args,
	}

	return sdk.client.Query(*sdk.identity, chaincode, peers)
}

func (sdk *sdkHandler) ListenEventFullBlock(peername, channelid string) (chan EventBlockResponse, error) {
	if peername == "" || channelid == "" {
		return nil, fmt.Errorf("ListenEventFullBlock peername or channelid is empty ")
	}
	ch := make(chan EventBlockResponse)
	ctx, cancel := context.WithCancel(context.Background())
	err := sdk.client.ListenForFullBlock(ctx, *sdk.identity, peername, channelid, ch)
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

func (sdk *sdkHandler) ListenEventFilterBlock(peername, channelid string) (chan EventBlockResponse, error) {
	if peername == "" || channelid == "" {
		return nil, fmt.Errorf("ListenEventFilterBlock peername or channelid is empty ")
	}
	ch := make(chan EventBlockResponse)
	ctx, cancel := context.WithCancel(context.Background())
	err := sdk.client.ListenForFilteredBlock(ctx, *sdk.identity, peername, channelid, ch)
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

func getChainCodeObj(args []string) (*ChainCode, error) {
	channelid := viper.GetString("other.channelId")
	chaincodeName := viper.GetString("other.chaincodeName")
	chaincodeVersion := viper.GetString("other.chaincodeVersion")
	mspId := viper.GetString("other.localMspId")
	if channelid == "" || chaincodeName == "" || chaincodeVersion == "" || mspId == "" {
		return nil, fmt.Errorf("channelid or ccname or ccver  or mspId is empty")
	}

	chaincode := ChainCode{
		ChannelId: channelid,
		Type:      ChaincodeSpec_GOLANG,
		Name:      chaincodeName,
		Version:   chaincodeVersion,
		Args:      args,
	}

	return &chaincode, nil
}

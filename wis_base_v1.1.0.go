package gohfc

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"context"

	"github.com/op/go-logging"
	"github.com/peersafe/gohfc/parseBlock"
)

type WisHandler struct {
	PeerConf         PeerConfig
	OrdererConf      OrdererConfig
	Mspids           string
	Pubkeys          string
	Prikeys          string
	CryptoFamilys    string
	Channeluuids     string
	ChaincodeName    string
	ChaincodeVersion string
	PeerName         string
	EventPeer        string
	OrderName        string
	Args             []string
	Ide              *Identity
	FaCli            *FabricClient
}

var wis_logger = logging.MustGetLogger("event")

func (w *WisHandler) Query() (*QueryResponse, error) {
	// 初始化
	err := w.Init()
	if err != nil {
		return nil, fmt.Errorf("Init Err : ", err.Error())
	}
	// 处理数据
	cc, err := w.getChainCodeObj()
	if err != nil {
		return nil, err
	}
	qRes, err := w.FaCli.Query(*w.Ide, *cc, []string{w.PeerName})
	if err != nil {
		wis_logger.Debug("Query Err = ", err.Error())
		return nil, err
	}
	return qRes[0], nil
}

func (w *WisHandler) Invoke() (*InvokeResponse, error) {

	err := w.Init()
	if err != nil {
		return nil, fmt.Errorf("Init Err : ", err.Error())
	}

	cc, err := w.getChainCodeObj()
	if err != nil {
		return nil, err
	}

	return w.FaCli.Invoke(*w.Ide, *cc, []string{w.PeerName}, w.OrderName)
}

func (w *WisHandler) ListenEventFullBlock(response chan<- EventBlockResponse) error {
	err := w.Init()
	if err != nil {
		return fmt.Errorf("Init Err : ", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	err = w.FaCli.ListenForFilteredBlock(ctx,*w.Ide,w.EventPeer,w.Channeluuids,response)
	if err != nil {
		cancel()
		return err
	}

	return nil
}

func (w *WisHandler) ListenForFullBlock(response chan<- parseBlock.Block) error {
	err := w.Init()
	if err != nil {
		return fmt.Errorf("Init Err : ", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	err = w.FaCli.ListenForFullBlock(ctx,*w.Ide,w.EventPeer,w.Channeluuids,response)
	if err != nil {
		cancel()
		return err
	}

	return nil
}

func (w *WisHandler) Init() error {
	format := logging.MustStringFormatter("%{shortfile} %{time:2006-01-02 15:04:05.000} [%{module}] %{level:.4s} : %{message}")
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)

	logging.SetBackend(backendFormatter).SetLevel(logging.DEBUG, "main")

	fabricClient :=  &FabricClient{}
	w.FaCli = fabricClient

	pubkey := findCurt(w.Pubkeys)
	prikey := findCurt(w.Prikeys)

	identity, err := LoadCertFromFile(pubkey, prikey)
	if err != nil {
		wis_logger.Error("LoadCertFromFile err = ", err)
		return err
	}
	identity.MspId = w.Mspids
	w.Ide = identity

	if "" != w.PeerName {
		peers := make(map[string]*Peer)
		peer, err := NewPeerFromConfig(w.PeerConf)
		if err != nil {
			return fmt.Errorf("Peer NewPeerFromConfig err :", err)
		}
		peers[w.PeerName] = peer
		w.FaCli.Peers = peers
	} else {
		wis_logger.Debug("This PeerName is empty!!!!")
	}


	if "" != w.OrderName {
		orderers := make(map[string]*Orderer)
		order, err := NewOrdererFromConfig(w.OrdererConf)
		if err != nil {
			return fmt.Errorf("Order NewOrdererFromConfig err :", err)
		}
		orderers[w.OrderName] = order
		w.FaCli.Orderers = orderers
	} else {
		wis_logger.Debug("This OrderName is empty!!!!")
	}

	if "" != w.EventPeer {
		eventpeers := make(map[string]*Peer)
		eventpeer, err := NewPeerFromConfig(w.PeerConf)
		if err != nil {
			return fmt.Errorf("EventPeer NewPeerFromConfig err :", err)
		}
		eventpeers[w.EventPeer] = eventpeer
		w.FaCli.EventPeers = eventpeers
	} else {
		wis_logger.Debug("This EventPeer is empty!!!!")
	}

	var crypto CryptoSuite
	switch w.CryptoFamilys {
	case "ecdsa":
		cryptoConfig := CryptoConfig{
			Family:    w.CryptoFamilys,
			Algorithm: "P256-SHA256",
			Hash:      "SHA2-256",
		}

		crypto, err = NewECCryptSuiteFromConfig(cryptoConfig)
		if err != nil {
			return err
		}

	case "gm":
		cryptoConfig := CryptoConfig{
			Family:    w.CryptoFamilys,
			Algorithm: "SM2-SM3",
			Hash:      "SM3",
		}

		crypto, err = NewECCryptSuiteFromConfig(cryptoConfig)
		if err != nil {
			return err
		}
	default:
		return ErrInvalidAlgorithmFamily
	}
	w.FaCli.Crypto = crypto

	return nil
}

func (w *WisHandler) getChainCodeObj() (*ChainCode, error) {
	channelid := w.Channeluuids
	chaincodeName := w.ChaincodeName
	chaincodeVersion := w.ChaincodeVersion
	mspId := w.Mspids
	if channelid == "" || chaincodeName == "" || chaincodeVersion == "" || mspId == "" {
		return nil, fmt.Errorf("channelid or ccname or ccver  or mspId is empty")
	}

	chaincode := ChainCode{
		ChannelId: channelid,
		Type:      ChaincodeSpec_GOLANG,
		Name:      chaincodeName,
		Version:   chaincodeVersion,
		Args:      w.Args,
	}

	return &chaincode, nil
}

func findCurt(path string) string {
	list, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println("ReadDir : ", err)
		fmt.Println(path)
		wis_logger.Debug(err.Error())
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

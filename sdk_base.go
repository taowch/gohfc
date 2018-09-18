package gohfc

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/peersafe/gohfc/parseBlock"
)

func QueryQscc(conf PeerConfig, cryptoFamily, pubkey, prikey, mspId, channeluuid string, args []string)(*QueryResponse, error) {
	return Query(conf, cryptoFamily, pubkey, prikey, mspId, channeluuid, "qscc", args)
}

func Query(conf PeerConfig, cryptoFamily, pubkey, prikey, mspId, channeluuid, queryChaincode string, args []string) (*QueryResponse, error) {
	peer, err := NewPeerFromConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("NewPeerFromConfig err :", err)
	}
	var crypto CryptoSuite
	switch cryptoFamily {
	case "ecdsa":
		cryptoConfig := CryptoConfig{
			Family:cryptoFamily,
			Algorithm:"P256-SHA256",
			Hash:"SHA2-256",
		}

		crypto, err = NewECCryptSuiteFromConfig(cryptoConfig)
		if err != nil {
			return nil, err
		}
	case "gm":
		cryptoConfig := CryptoConfig{
			Family:cryptoFamily,
			Algorithm:"SM2-SM3",
			Hash:"SM3",
		}

		crypto, err = NewECCryptSuiteFromConfig(cryptoConfig)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrInvalidAlgorithmFamily
	}

	chaincode := &ChainCode{
		ChannelId: channeluuid,
		Type:    ChaincodeSpec_GOLANG,
		Name:    queryChaincode,
		Args:    args,
	}

	findCert := func(path string) string {
		list, err := ioutil.ReadDir(path)
		if err != nil {
			fmt.Println("ReadDir : ", err)
			fmt.Println(path)
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

	pubkey = findCert(pubkey)
	prikey = findCert(prikey)

	identity, err := LoadCertFromFile(pubkey, prikey)
	if err != nil {
		return nil, err
	}

	identity.MspId = mspId

	prop, err := createTransactionProposal(*identity, *chaincode)
	if err != nil {
		return nil, err
	}
	proposal, err := signedProposal(prop.proposal, *identity, crypto)
	if err != nil {
		return nil, err
	}
	ch := make(chan *PeerResponse)
	go peer.Endorse(ch, proposal)
	peerResponse := <- ch
	response := &QueryResponse{
		Response: peerResponse.Response,
		Error:peerResponse.Err,
	}

	return response, nil
}

func  ListenEvent(conf PeerConfig, mspId, cryptoFamily, pubkey, prikey, channel string) (chan parseBlock.Block, error) {
	ch := make(chan parseBlock.Block)

	handler := &WisHandler{
		PeerConf:conf,
		Mspids:mspId,
		CryptoFamilys:cryptoFamily,
		EventPeer:conf.Host,
		Pubkeys:pubkey,
		Prikeys:prikey,
		Channeluuids:channel,
	}

	err := handler.ListenForFullBlock(ch)
	fmt.Println("newEventListener : ", err)
	if err != nil {
		return nil, err
	}
	return ch, nil
}
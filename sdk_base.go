package gohfc

import (
	"fmt"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/cendhu/fetch-block/src/events/parse"
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
	default:
		return nil, ErrInvalidAlgorithmFamily
	}

	channel := &Channel{
		ChannelName:channeluuid,
		MspId:mspId,
	}

	chaincode := &ChainCode{
		Channel: channel,
		Type:    ChaincodeSpec_GOLANG,
		Name:    queryChaincode,
		Args:    args,
	}

	findCert := func(path string) string {
		list, err := ioutil.ReadDir(path)
		if err != nil {
			fmt.Println("ReadDir : ", err)
			fmt.Println(path)
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

	pubkey = findCert(pubkey)
	prikey = findCert(prikey)

	identity, err := LoadCertFromFile(pubkey, prikey)
	if err != nil {
		return nil, err
	}

	prop, err := createTransactionProposal(identity, chaincode)
	if err != nil {
		return nil, err
	}
	proposal, err := signedProposal(prop.proposal, identity, crypto)
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

func  ListenEvent(conf PeerConfig, mspId, cryptoFamily, pubkey, prikey string) (chan parse.Block, error) {
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
	default:
		return nil, ErrInvalidAlgorithmFamily
	}

	findCert := func(path string) string {
		list, err := ioutil.ReadDir(path)
		if err != nil {
			fmt.Println("ReadDir : ", err)
			fmt.Println(path)
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

	pubkey = findCert(pubkey)
	prikey = findCert(prikey)

	identity, err := LoadCertFromFile(pubkey, prikey)
	if err != nil {
		return nil, err
	}

	ch := make(chan parse.Block)
	ctx, cancel := context.WithCancel(context.Background())

	err = newEventListener(ctx, ch, crypto, identity, mspId, peer)
	fmt.Println("newEventListener : ", err)
	if err != nil {
		cancel()
		return nil, err
	}
	return ch, nil
}

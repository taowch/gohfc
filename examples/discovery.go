package main

import (
	"github.com/peersafe/gohfc/discovery"
	discovery2 "github.com/hyperledger/fabric/protos/discovery"
	"github.com/hyperledger/fabric/cmd/common"
	"github.com/hyperledger/fabric/cmd/common/comm"
	"github.com/hyperledger/fabric/cmd/common/signer"
	"fmt"
)

func main(){
/*
	ConfigQueryType
	PeerMembershipQueryType
	ChaincodeQueryType
	LocalMembershipQueryType
*/
	d := discovery2.LocalMembershipQueryType
	Start_Discovery(d)
	return
}

func Start_Discovery(qType discovery2.QueryType){

	dis := discovery.NewRequest()
	conf := discovery.DisConfig{
		Addr: "peer0.org1.example.com:7051",
		Channel: "mychannel",
		CI: "/opt/gopath/src/github.com/hyperledger/fabric/examples/e2e_cli/crypto-config/peerOrganizations/org1.example.com/users/User1@org1.example.com/tls/client.key",
		CTLS: "/opt/gopath/src/github.com/hyperledger/fabric/examples/e2e_cli/crypto-config/peerOrganizations/org1.example.com/users/User1@org1.example.com/tls/client.crt",
		AddType: qType,
		CConfig: common.Config{
			Version: 0,
			TLSConfig:comm.Config{
				PeerCACertPath: "/opt/gopath/src/github.com/hyperledger/fabric/examples/e2e_cli/crypto-config/peerOrganizations/org1.example.com/users/User1@org1.example.com/tls/ca.crt",
				Timeout: 0,
			},
			SignerConfig: signer.Config{
				MSPID			: "Org1MSP",
				IdentityPath	: "/opt/gopath/src/github.com/hyperledger/fabric/examples/e2e_cli/crypto-config/peerOrganizations/org1.example.com/users/User1@org1.example.com/msp/signcerts/User1@org1.example.com-cert.pem",
				KeyPath			: "/opt/gopath/src/github.com/hyperledger/fabric/examples/e2e_cli/crypto-config/peerOrganizations/org1.example.com/users/User1@org1.example.com/msp/keystore/f795c66e7721acd49746e866828f062a7ce1b40c5e7643f80cd26143b516132d_sk",
			},
		},
		CCode: discovery2.ChaincodeInterest{
			Chaincodes:[]*discovery2.ChaincodeCall{
				{
					Name:"mycc",
				},
			},
		},
	}

	response ,err := dis.Add_Send2Service(conf)
	if err != nil {
		fmt.Println("Add_Send2Service Error = ",err)
		return
	}

	fmt.Println("response End!   response = ",response)


	err = dis.ParseResponse(conf,*response)
	if err != nil {
		fmt.Println("ParseResponse Error = ",err)
		return
	}
	return
}


package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/CognitionFoundry/gohfc"
)

const (
	ADM_PK = "/opt/gopath/src/github.com/hyperledger/fabric/examples/e2e_cli/crypto-config/peerOrganizations/org1.example.com/users/User1@org1.example.com/msp/admincerts/User1@org1.example.com-cert.pem"
	ADMSK  = "/opt/gopath/src/github.com/hyperledger/fabric/examples/e2e_cli/crypto-config/peerOrganizations/org1.example.com/users/User1@org1.example.com/msp/keystore/"
	MSP_ID = "Org1MSP"
)

var ADM_SK string

func main() {
	if get_sk(os.Args) != 0 {
		return
	}
	client, err := gohfc.NewFabricClient("./client.yaml")
	if err != nil {
		fmt.Printf("Error loading file: %v", err)
		os.Exit(1)
	}
	d, err := gohfc.NewDiscoveryFormConfig(nil)
	if err != nil {
		// 创建失败
		return
	}
	d.Crypto = client.Crypto
	d.Channel = "mychannel"
	d.CCName = "mycc"
	d.CollNames = ""

	d.MspId = MSP_ID
	d.CI = ADM_PK
	d.CTLS = "/opt/gopath/src/github.com/hyperledger/fabric/examples/e2e_cli/crypto-config/peerOrganizations/org1.example.com/users/User1@org1.example.com/tls/client.crt"

	identity, err := gohfc.LoadCertFromFile(ADM_PK, ADM_SK)
	if err != nil {
		fmt.Println("load cert from file failed:", err)
		return
	}
	identity.MspId = MSP_ID

	result, err := d.GetDiscoveryResult(*identity)
	if err != nil {
		fmt.Println("GetDiscoveryResult err = ", err)
	}

	fmt.Printf("%#v", result)
	fmt.Println(result.Config.ConfigResult.Orderers)
	for k, v := range result.PeersMem.Members.PeersByOrg {
		fmt.Println(k)
		fmt.Println(v.Peers)
	}

}

func get_sk(sk []string) int {

	if sk[1] == "" {
		return -1
	}
	var admsk string = ADMSK
	ADM_SK = admsk + sk[1]
	return 0
}

func LoadFileOrPanic(file string) []byte {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	return b
}

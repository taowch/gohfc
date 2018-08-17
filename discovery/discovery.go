package discovery

import (
	"github.com/peersafe/gohfc/discovery/client"
	discovery2 "github.com/peersafe/gohfc/discovery/cmd"
	discovery3 "github.com/hyperledger/fabric/protos/discovery"
	"io/ioutil"
	"github.com/hyperledger/fabric/cmd/common"
	"fmt"
	"os"
)

var err1 error

type Discovery struct {
	Req *discovery.Request
}

func NewRequest()*Discovery{
	dis := new(Discovery)
	dis.Req = discovery.NewRequest()
	return dis
}

func LoadFileOrPanic(file string) []byte {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	return b
}

type DisConfig struct {
	Addr 		string
	Channel 	string
	CI 			string
	CTLS 		string
	AddType 	discovery3.QueryType
	CConfig 	common.Config
	CCode		discovery3.ChaincodeInterest
}

// 添加 并发送
func (d *Discovery) Add_Send2Service(config DisConfig)(*discovery2.ServiceResponse,error){
	d.Req.OfChannel(config.Channel)
	CI := LoadFileOrPanic(config.CI)
	CTls := LoadFileOrPanic(config.CTLS)
	d.Req.Authentication = &discovery3.AuthInfo{
		ClientIdentity: 	CI,
		ClientTlsCertHash: CTls,
	}

	switch config.AddType {
	case discovery3.ConfigQueryType:
		c:=new(discovery2.ClientStub)
		d.Req = d.Req.AddConfigQuery()
		response,err := c.Send(config.Addr,config.CConfig,d.Req)
		return &response,err
	case discovery3.PeerMembershipQueryType:
		c := new(discovery2.ClientStub)
		d.Req = d.Req.AddPeersQuery()
		response,err := c.Send(config.Addr,config.CConfig,d.Req)
		return &response,err
	case discovery3.ChaincodeQueryType:
		r := new(discovery2.RawStub)
		d.Req,err1 = d.Req.AddEndorsersQuery(&config.CCode)
		response,err := r.Send(config.Addr,config.CConfig,d.Req)
		return &response,err
	case discovery3.LocalMembershipQueryType:
		c := new(discovery2.ClientStub)
		d.Req = d.Req.AddLocalPeersQuery()
		response,err := c.Send(config.Addr,config.CConfig,d.Req)
		return &response,err
	}

	fmt.Println( "AddType 传入错误!!!!!")
	return nil,nil

}


func (d *Discovery) ParseResponse( config DisConfig,response discovery2.ServiceResponse)error{

	switch config.AddType {
	case discovery3.ConfigQueryType:
		fmt.Println("====ConfigQueryType====")
		conf := &discovery2.ConfigResponseParser{
			os.Stdout,
		}
		err := conf.ParseResponse(config.Channel,response)
		if err != nil {
			return err
		}
	case discovery3.PeerMembershipQueryType:
		fmt.Println("====PeerMembershipQueryType====")
		peer := &discovery2.PeerResponseParser{
			os.Stdout,
		}
		err := peer.ParseResponse(config.Channel,response)
		if err != nil {
			return err
		}
	case discovery3.ChaincodeQueryType:
		fmt.Println("====ChaincodeQueryType====")
		ccQuery := &discovery2.EndorserResponseParser{
			os.Stdout,
		}
		err := ccQuery.ParseResponse(config.Channel,response)
		if err != nil {
			return err
		}
	case discovery3.LocalMembershipQueryType:
		fmt.Println("====LocalMembershipQueryType====")
		peer := &discovery2.PeerResponseParser{
			os.Stdout,
		}
		err := peer.ParseResponse(config.Channel,response)
		if err != nil {
			return err
		}
	}

	return nil
}

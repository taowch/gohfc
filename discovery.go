package gohfc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/discovery"
	"github.com/hyperledger/fabric/protos/msp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

var (
	configTypes = []discovery.QueryType{discovery.ConfigQueryType, discovery.PeerMembershipQueryType, discovery.ChaincodeQueryType, discovery.LocalMembershipQueryType}
)

type Discovery struct {
	Name   string
	Crypto CryptoSuite

	Addr   string
	caPath string
	Opts   []grpc.DialOption

	Channel   string
	MspId     string
	CI        string
	CTLS      string
	CCName    string
	CollNames string

	conn   *grpc.ClientConn
	client discovery.DiscoveryClient
}

type InvocationChain []*discovery.ChaincodeCall

// String returns a string representation of this invocation chain
func (ic InvocationChain) String() string {
	s, _ := json.Marshal(ic)
	return string(s)
}

// ValidateInvocationChain validates the InvocationChain's structure
func (ic InvocationChain) ValidateInvocationChain() error {
	if len(ic) == 0 {
		return errors.New("invocation chain should not be empty")
	}
	for _, cc := range ic {
		if cc.Name == "" {
			return errors.New("chaincode name should not be empty")
		}
	}
	return nil
}

type DisRequest struct {
	lastChannel            string
	lastIndex              int
	queryMapping           map[discovery.QueryType]map[string]int
	invocationChainMapping map[int][]InvocationChain
	*discovery.Request
}
type DisResponse struct {
	// 四个result struct
	Result *discovery.Response
	Err    error
}
type Result struct {
	Config      *discovery.QueryResult_ConfigResult
	ConfigErr   *discovery.QueryResult_Error
	CcQuery     *discovery.QueryResult_CcQueryRes
	CcQueryErr  *discovery.QueryResult_Error
	PeersMem    *discovery.QueryResult_Members
	PeersMemErr *discovery.QueryResult_Error
	//LocalMem		*discovery.LocalPeerQuery
	//LocalMemErr 	*discovery.QueryResult_Error
}

func (d *Discovery) GetDiscoveryResult(identity Identity) (*Result, error) {

	result := &Result{
		Config:      &discovery.QueryResult_ConfigResult{},
		ConfigErr:   &discovery.QueryResult_Error{},
		CcQuery:     &discovery.QueryResult_CcQueryRes{},
		CcQueryErr:  &discovery.QueryResult_Error{},
		PeersMem:    &discovery.QueryResult_Members{},
		PeersMemErr: &discovery.QueryResult_Error{},
	}
	CI := LoadFileOrPanic(d.CI)
	CTls := LoadFileOrPanic(d.CTLS)
	ci := &msp.SerializedIdentity{
		Mspid:   d.MspId,
		IdBytes: CI,
	}
	cliIde := MarshalOrPanic(ci)
	cc := make([]*discovery.ChaincodeCall, 0, 1)
	cc = append(cc, &discovery.ChaincodeCall{
		Name: d.CCName,
	})
	inter := &discovery.ChaincodeInterest{
		Chaincodes: cc,
	}

	fmt.Println("==============开始 Config==Get===============")
	req := d.NewDisRequest()
	req = req.AddConfigQuery()
	_, errResult := d.Discover(identity, result, req, &discovery.AuthInfo{
		ClientIdentity:    cliIde,
		ClientTlsCertHash: CTls,
	})
	fmt.Println("ConfigQueryType  err = ", errResult)
	fmt.Println("==============结束 Config==Get===============")

	fmt.Println("==============开始 PeersMem==Get===============")
	req = d.NewDisRequest()
	req = req.AddPeersMemQuery()
	_, errResult = d.Discover(identity, result, req, &discovery.AuthInfo{
		ClientIdentity:    cliIde,
		ClientTlsCertHash: CTls,
	})
	fmt.Println("PeerMembershipQueryType  err = ", errResult)

	fmt.Println("==============结束 PeersMem==Get===============")

	//fmt.Println("==============开始 LocalPeers==Get===============")
	//req = d.NewDisRequest()
	//req = req.AddLocalPeersQuery()
	//_,errResult =d.Discover(identity,result,req,&discovery.AuthInfo{
	//	ClientIdentity:cliIde,
	//	ClientTlsCertHash:CTls,
	//})
	//fmt.Println("LocalMembershipQueryType  err = ",errResult)
	//fmt.Println("==============结束 LocalPeers==Get===============")

	fmt.Println("==============开始 Endorsers==Get===============")
	req = d.NewDisRequest()
	var err error
	req, err = req.AddEndorsersQuery(inter)
	if err != nil {
		return result, err
	}
	_, errResult = d.Discover(identity, result, req, &discovery.AuthInfo{
		ClientIdentity:    cliIde,
		ClientTlsCertHash: CTls,
	})
	fmt.Println("ChaincodeQueryType  err = ", errResult)
	fmt.Println("==============结束 Endorsers==Get===============")

	return result, nil
}

func NewDiscoveryFormConfig(config *PeerConfig) (*Discovery, error) {
	d := Discovery{Addr: config.Host, caPath: config.TlsPath}
	if !config.UseTLS {
		d.Opts = []grpc.DialOption{grpc.WithInsecure()}
	} else if d.caPath != "" {
		careads, err := credentials.NewClientTLSFromFile(d.caPath, "")
		if err != nil {
			return nil, fmt.Errorf("cannot read DiscoveryConfig credentials err is: %v", err)
		}
		d.Opts = append(d.Opts, grpc.WithTransportCredentials(careads))
	}
	d.Opts = append(d.Opts,
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Duration(1) * time.Minute,
			Timeout:             time.Duration(20) * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxRecvMsgSize),
			grpc.MaxCallSendMsgSize(maxSendMsgSize)))
	return &d, nil
}
func (d *Discovery) NewDisRequest() *DisRequest {
	req := &DisRequest{
		lastChannel:            d.Channel,
		queryMapping:           make(map[discovery.QueryType]map[string]int),
		invocationChainMapping: make(map[int][]InvocationChain),
		Request:                &discovery.Request{},
	}
	for _, queryType := range configTypes {
		req.queryMapping[queryType] = make(map[string]int)
	}
	return req
}
func (d *DisRequest) addQueryMapping(queryType discovery.QueryType, key string) {
	d.queryMapping[queryType][key] = d.lastIndex
	d.lastIndex++
}
func (d *DisRequest) addChaincodeQueryMapping(invocationChains []InvocationChain) {
	d.invocationChainMapping[d.lastIndex] = invocationChains
}
func (req *DisRequest) AddConfigQuery() *DisRequest {
	ch := req.lastChannel
	q := &discovery.Query_ConfigQuery{
		ConfigQuery: &discovery.ConfigQuery{},
	}
	req.Queries = append(req.Queries, &discovery.Query{
		Channel: ch,
		Query:   q,
	})
	req.addQueryMapping(discovery.ConfigQueryType, ch)
	return req
}
func (d *DisRequest) AddEndorsersQuery(interests ...*discovery.ChaincodeInterest) (*DisRequest, error) {
	if err := validateInterests(interests...); err != nil {
		return nil, err
	}
	ch := d.lastChannel
	q := &discovery.Query_CcQuery{
		CcQuery: &discovery.ChaincodeQuery{
			Interests: interests,
		},
	}
	d.Queries = append(d.Queries, &discovery.Query{
		Channel: ch,
		Query:   q,
	})
	var invocationChains []InvocationChain
	for _, interest := range interests {
		invocationChains = append(invocationChains, interest.Chaincodes)
	}
	d.addChaincodeQueryMapping(invocationChains)
	d.addQueryMapping(discovery.ChaincodeQueryType, ch)
	return d, nil
}
func (d *DisRequest) AddLocalPeersQuery() *DisRequest {
	q := &discovery.Query_LocalPeers{
		LocalPeers: &discovery.LocalPeerQuery{},
	}
	d.Queries = append(d.Queries, &discovery.Query{
		Query: q,
	})
	d.addQueryMapping(discovery.LocalMembershipQueryType, "")
	return d
}
func (d *DisRequest) AddPeersMemQuery() *DisRequest {
	ch := d.lastChannel
	q := &discovery.Query_PeerQuery{
		PeerQuery: &discovery.PeerMembershipQuery{},
	}
	d.Queries = append(d.Queries, &discovery.Query{
		Channel: ch,
		Query:   q,
	})
	d.addQueryMapping(discovery.PeerMembershipQueryType, ch)
	return d
}
func (d *Discovery) Discover(identity Identity, resp *Result, req *DisRequest, auth *discovery.AuthInfo) (*DisResponse, error) {
	//	组织报文结构
	reqToBeSent := *req.Request
	reqToBeSent.Authentication = auth
	payload, err := proto.Marshal(&reqToBeSent)
	if err != nil {
		return nil, err
	}
	request, err := signedRequest(payload, &identity, d.Crypto)
	if err != nil {
		return nil, err
	}

	//	send
	r := d.send(request)
	//	解析

	for configType, channel2index := range req.queryMapping {
		switch configType {
		case discovery.ConfigQueryType:
			err = resp.mapConfig(channel2index, r)
		case discovery.ChaincodeQueryType:
			err = resp.mapEndorsers(channel2index, r, req.queryMapping, req.invocationChainMapping)
		case discovery.PeerMembershipQueryType:
			err = resp.mapPeerMembership(channel2index, r, discovery.PeerMembershipQueryType)
		//case discovery.LocalMembershipQueryType:
		//	err = resp.mapPeerMembership(channel2index, r, discovery.LocalMembershipQueryType)
		default:
		}
		//		fmt.Println(resp)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}
func signedRequest(prop []byte, identity *Identity, crypt CryptoSuite) (*discovery.SignedRequest, error) {
	sr, err := crypt.Sign(prop, identity.PrivateKey)
	if err != nil {
		return nil, err
	}
	return &discovery.SignedRequest{Payload: prop, Signature: sr}, nil
}
func (d *Discovery) send(prop *discovery.SignedRequest) *DisResponse {
	ch := make(chan *DisResponse)
	go d.Endorse(ch, prop)
	resp := make([]*DisResponse, 0, 1)
	for i := 0; i < 1; i++ {
		resp = append(resp, <-ch)
	}
	close(ch)
	return resp[0]
}
func (d *Discovery) Endorse(resp chan *DisResponse, prop *discovery.SignedRequest) {
	if d.conn == nil {
		conn, err := grpc.Dial(d.Addr, d.Opts...)
		if err != nil {
			resp <- &DisResponse{Result: nil, Err: err}
			return
		}
		d.conn = conn
		d.client = discovery.NewDiscoveryClient(d.conn)
	}

	proposalResp, err := d.client.Discover(context.Background(), prop)
	if err != nil {
		resp <- &DisResponse{Result: nil, Err: err}
		return
	}
	resp <- &DisResponse{Result: proposalResp, Err: err}
}
func MarshalOrPanic(pb proto.Message) []byte {
	data, err := proto.Marshal(pb)
	if err != nil {
		panic(err)
	}
	return data
}
func (resp Result) mapConfig(channel2index map[string]int, r *DisResponse) error {
	for _, index := range channel2index {
		config, err := r.ConfigAt(index)
		if config == nil && err == nil {
			return fmt.Errorf("expected QueryResult of either ConfigResult or Error but got %v instead", r.Result.Results[index])
		}

		if err != nil {
			resp.ConfigErr.Error.Content = err.Content
			continue
		}
		resp.Config.ConfigResult = config
	}
	return nil
}
func (resp Result) mapEndorsers(channel2index map[string]int, r *DisResponse, queryMapping map[discovery.QueryType]map[string]int, chaincodeQueryMapping map[int][]InvocationChain) error {
	for _, index := range channel2index {
		ccQueryRes, err := r.EndorsersAt(index)
		if ccQueryRes == nil && err == nil {
			return fmt.Errorf("expected QueryResult of either ChaincodeQueryResult or Error but got %v instead", r.Result.Results[index])
		}

		if err != nil {
			resp.CcQueryErr.Error.Content = err.Content
			continue
		}

		resp.CcQuery.CcQueryRes = ccQueryRes
	}
	return nil
}
func (resp Result) mapPeerMembership(channel2index map[string]int, r *DisResponse, qt discovery.QueryType) error {
	for _, index := range channel2index {
		membersRes, err := r.MembershipAt(index)
		if membersRes == nil && err == nil {
			return fmt.Errorf("expected QueryResult of either PeerMembershipResult or Error but got %v instead", r.Result.Results[index])
		}

		if err != nil {
			resp.PeersMemErr.Error.Content = err.Content
			continue
		}
		resp.PeersMem.Members = membersRes
	}
	return nil
}

func validateInterests(interests ...*discovery.ChaincodeInterest) error {
	if len(interests) == 0 {
		return errors.New("no chaincode interests given")
	}
	for _, interest := range interests {
		if interest == nil {
			return errors.New("chaincode interest is nil")
		}
		if err := InvocationChain(interest.Chaincodes).ValidateInvocationChain(); err != nil {
			return err
		}
	}
	return nil
}
func (m *DisResponse) ConfigAt(i int) (*discovery.ConfigResult, *discovery.Error) {
	r := m.Result.Results[i]
	return r.GetConfigResult(), r.GetError()
}
func (m *DisResponse) MembershipAt(i int) (*discovery.PeerMembershipResult, *discovery.Error) {
	r := m.Result.Results[i]
	return r.GetMembers(), r.GetError()
}
func (m *DisResponse) EndorsersAt(i int) (*discovery.ChaincodeQueryResult, *discovery.Error) {
	r := m.Result.Results[i]
	return r.GetCcQueryRes(), r.GetError()
}
func LoadFileOrPanic(file string) []byte {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	return b
}

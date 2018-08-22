# /bin/bash

SK_PWD=`ls /opt/gopath/src/github.com/hyperledger/fabric/examples/e2e_cli/crypto-config/peerOrganizations/org1.example.com/users/User1@org1.example.com/msp/keystore`

echo $SK_PWD

go clean

go build
./examples $SK_PWD 



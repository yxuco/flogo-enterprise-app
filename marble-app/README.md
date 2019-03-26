# marble-app
This is a the sample chaincode, [marbles02](https://github.com/hyperledger/fabric-samples/tree/release-1.4/chaincode/marbles02) for [Hyperledger Fabric](https://www.hyperledger.org/projects/fabric) implemented by using a [TIBCO Flogo® Enterprise](https://docs.tibco.com/products/tibco-flogo-enterprise-2-4-0) model.

## Prerequisite
- Download [TIBCO Flogo® Enterprise 2.4](https://edelivery.tibco.com/storefront/eval/tibco-flogo-enterprise/prod11810.html)
- [Install Go](https://golang.org/doc/install)
- Clone [Hyperledger Fabric Samples](https://github.com/hyperledger/fabric-samples)
- Clone [This Flogo extension](https://github.com/yxuco/flogo-enterprise-app)

## Edit smart contract
- Start TIBCO Flogo® Enterprise as described in [User's Guide](https://docs.tibco.com/pub/flogo/2.4.0/doc/pdf/TIB_flogo_2.4_users_guide.pdf?id=1)
- Upload [`fabticExtension.zip`](https://github.com/yxuco/flogo-enterprise-app/blob/master/fabricExtension.zip) to TIBCO Flogo® Enterprise [Extensions](http://localhost:8090/wistudio/extensions).  Note that you can recreate this `zip` by using the script [`zip-fabric.sh`](https://github.com/yxuco/flogo-enterprise-app/blob/master/zip-fabric.sh)
- Create new Flogo App of name `marble_app` and choose `Import app` to import the model [`marble_app.json`](https://github.com/yxuco/flogo-enterprise-app/blob/master/marble-app/marble_app.json)
- You can then add or update contract transactions using the graphical modeler of the TIBCO Flogo® Enterprise.

## Build and deploy chaincode to Hyperledger Fabric
- Export the Flogo App, and copy the downloaded model file, i.e., [`marble_app.json`](https://github.com/yxuco/flogo-enterprise-app/blob/master/marble-app/marble_app.json) to folder `marble-app`.  You can skip this step if you did not modify the app in Flogo® Enterprise.
- In the `marble-app` folder, execute `make create` to generate source code for the chaincode.  This step downloads all dependent packages, and thus may take a while depending on the network speed.
- Execute `make deploy` to deploy the chaincode to the `fabric-samples` chaincode folder.  Note: you may need to edit the [`Makefile`](https://github.com/yxuco/flogo-enterprise-app/blob/master/marble-app/Makefile) and set `CC_DEPLOY` to match the installation folder of `fabric-samples` if it is not downloaded to the default location under `$GOPATH`.

## Test smart contract
Start Hyperledger Fabric test network in dev mode:
```
cd $GOPATH//src/github.com/hyperledger/fabric-samples/chaincode-docker-devmode
docker-compose -f docker-compose-simple.yaml up
```
In another terminal, start the chaincode:
```
docker exec -it chaincode bash
cd marble_cc
CORE_PEER_ADDRESS=peer:7052 CORE_CHAINCODE_ID_NAME=marble_cc:0 CORE_CHAINCODE_LOGGING_LEVEL=DEBUG ./marble_cc
```
In a third terminal, install chaincode and send tests:
```
docker exec -it cli bash
peer chaincode install -p chaincodedev/chaincode/marble_cc -n marble_cc -v 0
peer chaincode instantiate -n marble_cc -v 0 -c '{"Args":["init"]}' -C myc

# test transactions using the following commands:
peer chaincode invoke -C myc -n marble_cc -c '{"Args":["initMarble","marble1","blue","35","tom"]}'
peer chaincode invoke -C myc -n marble_cc -c '{"Args":["initMarble","marble2","red","50","tom"]}'
peer chaincode invoke -C myc -n marble_cc -c '{"Args":["initMarble","marble3","blue","70","tom"]}'
peer chaincode invoke -C myc -n marble_cc -c '{"Args":["transferMarble","marble2","jerry"]}'
peer chaincode query -C myc -n marble_cc -c '{"Args":["readMarble","marble2"]}'
peer chaincode query -C myc -n marble_cc -c '{"Args":["getMarblesByRange","marble1","marble3"]}'
peer chaincode invoke -C myc -n marble_cc -c '{"Args":["transferMarblesBasedOnColor","blue","jerry"]}'
peer chaincode query -C myc -n marble_cc -c '{"Args":["getHistoryForMarble","marble1"]}'
peer chaincode invoke -C myc -n marble_cc -c '{"Args":["delete","marble1"]}'
peer chaincode query -C myc -n marble_cc -c '{"Args":["getHistoryForMarble","marble1"]}'
```
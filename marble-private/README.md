# marble-private
This is a the sample chaincode using private collections and transient transaction arguments, [marbles02_private](https://github.com/hyperledger/fabric-samples/tree/release-1.4/chaincode/marbles02_private) for [Hyperledger Fabric](https://www.hyperledger.org/projects/fabric) implemented by using a [TIBCO Flogo® Enterprise](https://docs.tibco.com/products/tibco-flogo-enterprise-2-4-0) model.

## Prerequisite
- Download [TIBCO Flogo® Enterprise 2.4](https://edelivery.tibco.com/storefront/eval/tibco-flogo-enterprise/prod11810.html)
- [Install Go](https://golang.org/doc/install)
- Clone [Hyperledger Fabric Samples](https://github.com/hyperledger/fabric-samples)
- Clone [This Flogo extension](https://github.com/yxuco/flogo-enterprise-app)

## Edit smart contract
- Start TIBCO Flogo® Enterprise as described in [User's Guide](https://docs.tibco.com/pub/flogo/2.4.0/doc/pdf/TIB_flogo_2.4_users_guide.pdf?id=1)
- Upload [`fabticExtension.zip`](https://github.com/yxuco/flogo-enterprise-app/blob/master/fabricExtension.zip) to TIBCO Flogo® Enterprise [Extensions](http://localhost:8090/wistudio/extensions).  Note that you can recreate this `zip` by using the script [`zip-fabric.sh`](https://github.com/yxuco/flogo-enterprise-app/blob/master/zip-fabric.sh)
- Create new Flogo App of name `marble_private` and choose `Import app` to import the model [`marble_private.json`](https://github.com/yxuco/flogo-enterprise-app/blob/master/marble-private/marble_private.json)
- You can then add or update contract transactions using the graphical modeler of the TIBCO Flogo® Enterprise.

## Build and deploy chaincode to Hyperledger Fabric
- Export the Flogo App, and copy the downloaded model file, i.e., [`marble_private.json`](https://github.com/yxuco/flogo-enterprise-app/blob/master/marble-private/marble_private.json) to folder `marble-private`.  You can skip this step if you did not modify the app in Flogo® Enterprise.
- In the `marble-private` folder, execute `make create` to generate source code for the chaincode.  This step downloads all dependent packages, and thus may take a while depending on the network speed.
- Execute `make deploy` to deploy the chaincode to the `fabric-samples` chaincode folder.  Note: you may need to edit the [`Makefile`](https://github.com/yxuco/flogo-enterprise-app/blob/master/marble-private/Makefile) and set `CC_DEPLOY` to match the installation folder of `fabric-samples` if it is not downloaded to the default location under `$GOPATH`.

## Test smart contract

Similar to the [marble-app](https://github.com/yxuco/flogo-enterprise-app/blob/master/marble-app) sample, except that it requires setting up private collections.
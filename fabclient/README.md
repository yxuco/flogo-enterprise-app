# Flogo extension for Hyperledger Fabric client

This Flogo extension is designed to allow delopers to use the zero-code visual programming environment of the TIBCO Flogo® to design and implement client apps that interacts with a Hyperledger Fabric network.  This extension supports the following release versions:
- [TIBCO Flogo® Enterprise 2.4](https://docs.tibco.com/products/tibco-flogo-enterprise-2-4-0)
- [Hyperledger Fabric 1.4](https://www.hyperledger.org/projects/fabric)
- [Hyperledger Fabric Go SDK v1.0.0-alpha5](https://github.com/hyperledger/fabric-sdk-go)

The [Fabric Connector](https://github.com/yxuco/flogo-enterprise-app/tree/master/fabclient/connector/fabconnector) allows you to configure the target fabric network used by the app.

It supports the following activity for submitting an `invoke` or `query` request to a Hyperledger Fabric network.
- [Fabric Request](https://github.com/yxuco/flogo-enterprise-app/tree/master/fabclient/activity/fabrequest): Send an `invoke` request to execute a specified chaincode that inserts or updates records in the distributed ledger or private collections.  Send a `query` request to execute a specified chaincode that queries the records in the distributed ledger or private collections without changing any state.

More activities can be added to integrate with blockchain events.

With these extensions, Hyperledger Fabric client apps can be designed and implemented with zero code. Refer to the sample [`marble-client`](https://github.com/yxuco/flogo-enterprise-app/tree/master/marble-client) for more details of a REST service implemented with `TIBCO Flogo® Enterprise` that updates and retrieves data on a distributed ledger of Hyperledger Fabric.
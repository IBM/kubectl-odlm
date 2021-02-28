# kubectl-odlm
browse Operand Deployment Lifecycle Manager resources from the command line

## How to use

```
make build
./bin/kubectl-odlm tree <OperandRequest Name>
```

Example:
```
./bin/kubectl-odlm tree common-service

NAMESPACE            NAME                                                     
ibm-common-services  OperandRequest/common-service                            
ibm-common-services  ├─Subscription/ibm-iam-operator                          
ibm-common-services  │ └─ClusterServiceVersion/ibm-iam-operator.v3.9.1        
ibm-common-services  │   ├─OIDCClientWatcher/example-oidcclientwatcher        
ibm-common-services  │   ├─PolicyDecision/example-policydecision              
ibm-common-services  │   ├─Authentication/example-authentication              
ibm-common-services  │   ├─Pap/example-pap                                    
ibm-common-services  │   ├─PolicyController/policycontroller-deployment       
ibm-common-services  │   ├─SecretWatcher/secretwatcher-deployment             
ibm-common-services  │   ├─SecurityOnboarding/example-securityonboarding      
ibm-common-services  │   ├─OperandRequest/ibm-iam-request                     
ibm-common-services  │   └─OperandBindInfo/ibm-iam-bindinfo                   
ibm-common-services  └─Subscription/ibm-healthcheck-operator                  
ibm-common-services    └─ClusterServiceVersion/ibm-healthcheck-operator.v3.9.0
ibm-common-services      ├─HealthService/system-healthcheck-service           
ibm-common-services      ├─MustGatherConfig/must-gather-common-service-config 
ibm-common-services      ├─MustGatherConfig/must-gather-default-config        
ibm-common-services      └─MustGatherService/must-gather-service              
```

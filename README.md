# Sample Controller for API Server

This sample controller will create a deployment when a new Custom Resource is created.
All the CR and Deployment will create on a custom namespace called `apiserver`. So create
the namespace first.
```shell
kubectl create ns apiserver
```

### Create Custom CRD

```shell
kubectl apply -f artifacts/crd.yaml
```

### Command to run Controller
```shell
go run . #Run the controller code(Terminal 1)
kubectl apply -f artifacts/apiserver.yaml #Create the custom resource
# As the CR is created so the corresponding deployment should be created
#Check them using the following command
kubectl get deployment --namespace=apiserver
#Update the CR Yaml file and apply it. The deployment will change accordingly
#Delete the CR 
kubectl delete -f artifacts/apiserver.yaml
#Corresponding Deployment also be deleted by this time
```
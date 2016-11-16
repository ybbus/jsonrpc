# jsonrpc
A go implementation of json-rpc over http.

## Examples

### Simple call

Let's start by executing a simple json-rpc http call:

```go
func main() {
    rpcClient := NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("addNumbers", 1, 2)
}
```

### ID Autoincrement

Per default the ID of the json-rpc request increments automatically for each request.
You can change this behaviour:

```go
func main() {
    rpcClient := NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("addNumbers", 1, 2) // send with ID == 0
    response, _ = rpcClient.Call("addNumbers", 1, 2) // send with ID == 1
    rpcClient.SetNextID(10)
    response, _ = rpcClient.Call("addNumbers", 1, 2) // send with ID == 10
    rpcClient.SetAutoIncrementID(false)
    response, _ = rpcClient.Call("addNumbers", 1, 2) // send with ID == 11
    response, _ = rpcClient.Call("addNumbers", 1, 2) // send with ID == 11
}
```

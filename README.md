# jsonrpc
A go implementation of json-rpc over http.

## Examples

Let's start by executing a simple json-rpc http call:

```go
func main() {
    rpcClient := NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("addNumbers", 1, 2)
}
```
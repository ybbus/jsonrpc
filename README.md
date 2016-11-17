# jsonrpc
A go implementation of json-rpc over http. This implementation only provides positional parameters (array of arbitrary types).

## Examples

### Simple call

Let's start by executing a simple json-rpc http call:

```go
func main() {
    rpcClient := NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("getDate")
    // generates body: {"jsonrpc":"2.0","method":"getDate","id":0}
}
```

Call a function with parameter:

```go
func main() {
    rpcClient := NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("addNumbers", 1, 2)
    // generates body: {"jsonrpc":"2.0","method":"addNumbers","params":[1,2],"id":0}
}
```

Call a function with arbitrary parameters:

```go
func main() {
    rpcClient := NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("createPerson", "Alex", 33, "Germany")
    // generates body: {"jsonrpc":"2.0","method":"createPerson","params":["Alex",33,"Germany"],"id":0}
}
```

Call a function providing custom data structures as parameters:

```go
type Person struct {
  Name    string `json:"name"`
  Age     int `json:"age"`
  Country string `json:"country"`
}
func main() {
    rpcClient := NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("createPerson", Person{"Alex", 33, "Germany"})
    // generates body: {"jsonrpc":"2.0","method":"createPerson","params":[{"name":"Alex","age":33,"country":"Germany"}],"id":0}
}
```

Complex example:

```go
type Person struct {
  Name    string `json:"name"`
  Age     int `json:"age"`
  Country string `json:"country"`
}
func main() {
    rpcClient := NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("createPersonsWithRole", []Person{{"Alex", 33, "Germany"}, {"Barney", 38, "Germany"}}, []string{"Admin", "User"})
    // generates body: {"jsonrpc":"2.0","method":"createPersonsWithRole","params":[[{"name":"Alex","age":33,"country":"Germany"},{"name":"Barney","age":38,"country":"Germany"}],["Admin","User"]],"id":0}
}
```

### Basic authentication

If the rpc-service is running behind a basic authentication you can easily set the credentials:

```go
func main() {
    rpcClient := NewRPCClient("http://my-rpc-service:8080/rpc")
    rpcClient.SetBasicAuth("alex", "secret")
    response, _ := rpcClient.Call("addNumbers", 1, 2) // send with Authorization-Header
}
```

### Set custom headers

Setting some custom headers (e.g. when using another authentication) is simple:
```go
func main() {
    rpcClient := NewRPCClient("http://my-rpc-service:8080/rpc")
    rpcClient.SetCustomHeader("Authorization", "Bearer abcd1234")
    response, _ := rpcClient.Call("addNumbers", 1, 2) // send with a custom Auth-Header
}
```

### ID auto-increment

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

### Set a custom httpClient

If you have some special needs on the http.Client of the standard go library, just provide your own one.
For example to use a proxy when executing json-rpc calls:

```go
func main() {
    rpcClient := NewRPCClient("http://my-rpc-service:8080/rpc")

    proxyURL, _ := url.Parse("http://proxy:8080")
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}

	httpClient := &http.Client{
		Transport: transport,
	}

	rpcClient.SetHTTPClient(httpClient)
}
```
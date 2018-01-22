[![Go Report Card](https://goreportcard.com/badge/github.com/ybbus/jsonrpc)](https://goreportcard.com/report/github.com/ybbus/jsonrpc)
[![GoDoc](https://godoc.org/github.com/ybbus/jsonrpc?status.svg)](https://godoc.org/github.com/ybbus/jsonrpc)
[![GitHub license](https://img.shields.io/github/license/mashape/apistatus.svg)]()

# JSON-RPC 2.0 Client for golang
A go implementation of an rpc client using json as data format over http.
The implementation is based on the JSON-RPC 2.0 specification: http://www.jsonrpc.org/specification

Supports:
- requests with arbitrary parameters
- requests with named parameters
- notifications
- batch requests
- convenient response retrieval
- basic authentication
- custom headers
- custom http client

## Installation

```sh
go get -u github.com/ybbus/jsonrpc
```

## Getting started
Let's say we want to retrieve a person with a specific id using rpc-json over http.
Then we want to save this person with new properties.
We have to provide basic authentication credentials.
(Error handling is omitted here)

```go
type Person struct {
    Id   int `json:"id"`
    Name string `json:"name"`
    Age  int `json:"age"`
}

func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    rpcClient.SetBasicAuth("alex", "secret")

    response, _ := rpcClient.Call("getPersonById", 123)

    person := Person{}
    response.GetObject(&person)

    person.Age = 33
    rpcClient.Call("updatePerson", person)
}
```

## In detail

### Generating rpc-json requests

Let's start by executing a simple json-rpc http call:
In production code: Always make sure to check err != nil first!

This calls generate and send a valid rpc-json object. (see: http://www.jsonrpc.org/specification#request_object)

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("getDate")
    // generates body: {"jsonrpc":"2.0","method":"getDate","id":0}
}
```

Call a function with parameter:

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("addNumbers", 1, 2)
    // generates body: {"jsonrpc":"2.0","method":"addNumbers","params":[1,2],"id":0}
}
```

Call a function with arbitrary parameters:

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("createPerson", "Alex", 33, "Germany")
    // generates body: {"jsonrpc":"2.0","method":"createPerson","params":["Alex",33,"Germany"],"id":0}
}
```

Call a function with named parameters:

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
		rpcClient.CallNamed("createPerson", map[string]interface{}{
		"name":      "Bartholomew Allen",
		"nicknames": []string{"Barry", "Flash",},
		"male":      true,
		"age":       28,
		"address":   map[string]interface{}{"street": "Main Street", "city": "Central City"},
	})
    // generates body: {"jsonrpc":"2.0","method":"createPerson","params":
	//	{"name": "Bartholomew Allen", "nicknames": ["Barry", "Flash"], "male": true, "age": 28,
	//	"address": {"street": "Main Street", "city": "Central City"}}
	//	,"id":0}
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
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
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
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("createPersonsWithRole", []Person{{"Alex", 33, "Germany"}, {"Barney", 38, "Germany"}}, []string{"Admin", "User"})
    // generates body: {"jsonrpc":"2.0","method":"createPersonsWithRole","params":[[{"name":"Alex","age":33,"country":"Germany"},{"name":"Barney","age":38,"country":"Germany"}],["Admin","User"]],"id":0}
}
```

### Notification

A jsonrpc notification is a rpc call to the server without expecting a response.
Only an error object is returned in case of networkt / http error.
No id field is set in the request json object. (see: http://www.jsonrpc.org/specification#notification)

Execute an simple notification:

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    err := rpcClient.Notification("disconnectClient", 123)
    if err != nil {
        //error handling goes here
    }
}
```

### Batch rpcjson calls

A jsonrpc batch call encapsulates multiple json-rpc requests in a single rpc-service call.
It returns an array of results (for all non-notification requests).
(see: http://www.jsonrpc.org/specification#batch)

Execute two jsonrpc calls and a single notification as batch:

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient(httpServer.URL)

	req1 := rpcClient.NewRPCRequestObject("addNumbers", 1, 2)
	req2 := rpcClient.NewRPCRequestObject("getPersonByName", "alex")
	notify1 := rpcClient.NewRPCNotificationObject("disconnect", true)

	responses, _ := rpcClient.Batch(req1, req2, notify1)

    person := Person{}
    response2, _ := responses.GetResponseOf(req2)
    response2.GetObject(&person)
}
```

To update the ID of an existing rpcRequest object:
```go
func main() {
    rpcClient := jsonrpc.NewRPCClient(httpServer.URL)

	req1 := rpcClient.NewRPCRequestObject("addNumbers", 1, 2)
	req2 := rpcClient.NewRPCRequestObject("getPersonByName", "alex")
	notify1 := rpcClient.NewRPCNotifyObject("disconnect", true)

	responses, _ := rpcClient.Batch(req1, req2, notify1)

    rpcClient.UpdateRequestID(req1) // updates id to the next valid id if autoincrement is enabled
}
```

### Working with rpc-json responses


Before working with the response object, make sure to check err != nil first.
This error indicates problems on the network / http level of an error when parsing the json response.

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    response, err := rpcClient.Call("addNumbers", 1, 2)
    if err != nil {
        //error handling goes here
    }
}
```

The next thing you have to check is if an rpc-json protocol error occoured. This is done by checking if the Error field in the rpc-response != nil:
(see: http://www.jsonrpc.org/specification#error_object)

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    response, err := rpcClient.Call("addNumbers", 1, 2)
    if err != nil {
        //error handling goes here
    }

    if response.Error != nil {
        // check response.Error.Code, response.Error.Message, response.Error.Data  here
    }
}
```

After making sure that no errors occoured you can now examine the RPCResponse object.
When executing a json-rpc request, most of the time you will be interested in the "result"-property of the returned json-rpc response object.
(see: http://www.jsonrpc.org/specification#response_object)
The library provides some helper functions to retrieve the result in the data format you are interested in.
Again: check for err != nil here to be sure the expected type was provided in the response and could be parsed.

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("addNumbers", 1, 2)

    result, err := response.GetInt()
    if err != nil {
        // result seems not to be an integer value
    }

    // helpers provided for all primitive types:
    response.GetInt() // int32 or int64 depends on architecture / implementation
    response.GetInt64()
    response.GetFloat64()
    response.GetString()
    response.GetBool()
}
```

Retrieving arrays and objects is also very simple:

```go
// json annotations are only required to transform the structure back to json
type Person struct {
    Id   int `json:"id"`
    Name string `json:"name"`
    Age  int `json:"age"`
}

func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("getPersonById", 123)

    person := Person{}
    err := response.GetObject(&Person) // expects a rpc-object result value like: {"id": 123, "name": "alex", "age": 33}

    fmt.Println(person.Name)
}
```

Retrieving arrays e.g. of ints:

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    response, _ := rpcClient.Call("getRandomNumbers", 10)

    rndNumbers := []int{}
    err := response.getObject(&rndNumbers) // expects a rpc-object result value like: [10, 188, 14, 3]

    fmt.Println(rndNumbers[0])
}
```

### Basic authentication

If the rpc-service is running behind a basic authentication you can easily set the credentials:

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    rpcClient.SetBasicAuth("alex", "secret")
    response, _ := rpcClient.Call("addNumbers", 1, 2) // send with Authorization-Header
}
```

### Set custom headers

Setting some custom headers (e.g. when using another authentication) is simple:
```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
    rpcClient.SetCustomHeader("Authorization", "Bearer abcd1234")
    response, _ := rpcClient.Call("addNumbers", 1, 2) // send with a custom Auth-Header
}
```

### ID auto-increment

Per default the ID of the json-rpc request increments automatically for each request.
You can change this behaviour:

```go
func main() {
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")
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
    rpcClient := jsonrpc.NewRPCClient("http://my-rpc-service:8080/rpc")

    proxyURL, _ := url.Parse("http://proxy:8080")
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}

	httpClient := &http.Client{
		Transport: transport,
	}

	rpcClient.SetHTTPClient(httpClient)
}
```

package jsonrpc

import (
	"fmt"
)

func Example() {
	type Person struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		Country string `json:"country"`
	}

	// create client
	rpcClient := NewClient("http://my-rpc-service")

	// execute rpc to service
	response, _ := rpcClient.Call("getPersonByID", 12345)

	// parse result into struct
	var person Person
	response.GetObject(&person)

	// change result and send back using rpc
	person.Age = 35
	rpcClient.Call("setPersonByID", 12345, person)
}

func ExampleRPCClient_Call() {
	rpcClient := NewClient("http://my-rpc-service")

	// result processing omitted, see: RPCResponse methods
	rpcClient.Call("getTimestamp")

	rpcClient.Call("getPerson", 1234)

	rpcClient.Call("addNumbers", 5, 2, 3)

	rpcClient.Call("strangeFunction", 1, true, "alex", 3.4)

	type Person struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		Country string `json:"country"`
	}

	person := Person{
		Name:    "alex",
		Age:     33,
		Country: "germany",
	}

	rpcClient.Call("setPersonByID", 123, person)
}

func ExampleRPCResponse() {
	rpcClient := NewClient("http://my-rpc-service")

	response, _ := rpcClient.Call("addNumbers", 1, 2, 3)
	sum, _ := response.GetInt()
	fmt.Println(sum)

	response, _ = rpcClient.Call("isValidEmail", "my@ema.il")
	valid, _ := response.GetBool()
	fmt.Println(valid)

	response, _ = rpcClient.Call("getJoke")
	joke, _ := response.GetString()
	fmt.Println(joke)

	response, _ = rpcClient.Call("getPi", 10)
	piRounded, _ := response.GetFloat64()
	fmt.Println(piRounded)

	var rndNumbers []int
	response, _ = rpcClient.Call("getRndIntNumbers", 10)
	response.GetObject(&rndNumbers)
	fmt.Println(rndNumbers[0])

	type Person struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		Country string `json:"country"`
	}

	var p Person
	response, _ = rpcClient.Call("getPersonByID", 1234)
	response.GetObject(&p)
	fmt.Println(p.Name)
}


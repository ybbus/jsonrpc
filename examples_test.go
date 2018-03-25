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
	rpcClient := NewRPCClient("http://my-rpc-service")

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
	rpcClient := NewRPCClient("http://my-rpc-service")

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

func ExampleRPCClient_CallNamed() {
	rpcClient := NewRPCClient("http://my-rpc-service")

	// result processing omitted, see: RPCResponse methods
	rpcClient.CallNamed("createPerson", map[string]interface{}{
		"name":      "Bartholomew Allen",
		"nicknames": []string{"Barry", "Flash"},
		"male":      true,
		"age":       28,
		"address":   map[string]interface{}{"street": "Main Street", "city": "Central City"},
	})
}

func ExampleRPCClient_CallObject() {
	rpcClient := NewRPCClient("http://my-rpc-service")

	type Person struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		Country string `json:"country"`
	}

	// CallObject only handles (pointer of) structs
	rpcClient.CallObject("createPerson", Person{
		Name:    "Alex",
		Age:     35,
		Country: "Germany",
	})

	// CallObject only handles (pointer of) structs
	rpcClient.CallObject("createPerson", &Person{
		Name:    "Alex",
		Age:     35,
		Country: "Germany",
	})

	// don't forget the json tags on anonymous structs
	rpcClient.CallObject("createPerson", struct {
		Length float32 `json:"length"`
		Height float32 `json:"height"`
	}{
		Length: 5.6,
		Height: 7.8,
	})

	// Everything that is not a struct returns an error
	_, err := rpcClient.CallObject("setBirthday", "26.06.2013")
	fmt.Println(err.Error())

	// Only other value that is allowed is nil
	rpcClient.CallObject("doNothing", nil)
}

func ExampleRPCResponse() {
	rpcClient := NewRPCClient("http://my-rpc-service")

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

func ExampleRPCClient_Batch() {
	rpcClient := NewRPCClient(httpServer.URL)

	req1 := rpcClient.NewRPCRequestObject("addNumbers", 1, 2, 3)
	req2 := rpcClient.NewRPCRequestObject("getTimestamp")
	responses, _ := rpcClient.Batch(req1, req2)

	response, _ := responses.GetResponseOf(req2)
	timestamp, _ := response.GetInt()

	fmt.Println(timestamp)
}

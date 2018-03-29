package jsonrpc

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

// needed to retrieve requests that arrived at httpServer for further investigation
var requestChan = make(chan *RequestData, 1)

// the request datastructure that can be retrieved for test assertions
type RequestData struct {
	request *http.Request
	body    string
}

// set the response body the httpServer should return for the next request
var responseBody = ""

var httpServer *httptest.Server

// start the testhttp server and stop it when tests are finished
func TestMain(m *testing.M) {
	httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		// put request and body to channel for the client to investigate them
		requestChan <- &RequestData{r, string(data)}

		fmt.Fprintf(w, responseBody)
	}))
	defer httpServer.Close()

	os.Exit(m.Run())
}

func TestSimpleRpcCallHeaderCorrect(t *testing.T) {
	RegisterTestingT(t)

	rpcClient := NewClient(httpServer.URL)
	rpcClient.Call("add", 1, 2)

	req := (<-requestChan).request

	Expect(req.Method).To(Equal("POST"))
	Expect(req.Header.Get("Content-Type")).To(Equal("application/json"))
	Expect(req.Header.Get("Accept")).To(Equal("application/json"))
}

// test if the structure of an rpc request is built correctly validate the data that arrived on the server
func TestRpcClient_Call(t *testing.T) {
	RegisterTestingT(t)
	rpcClient := NewClient(httpServer.URL)

	person := Person{
		Name:    "Alex",
		Age:     35,
		Country: "Germany",
	}

	food := Drink{
		Name:        "Cuba Libre",
		Ingredients: []string{"rum", "cola"},
	}

	rpcClient.Call("nullParam", nil)
	body := (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"nullParam","params":[null],"id":1}`))

	rpcClient.Call("nullParams", nil, nil)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"nullParams","params":[null,null],"id":1}`))

	rpcClient.Call("boolParam", true)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"boolParam","params":[true],"id":1}`))

	rpcClient.Call("boolParams", true, false, true)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"boolParams","params":[true,false,true],"id":1}`))

	rpcClient.Call("stringParam", "Alex")
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"stringParam","params":["Alex"],"id":1}`))

	rpcClient.Call("stringParams", "JSON", "RPC")
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"stringParams","params":["JSON","RPC"],"id":1}`))

	rpcClient.Call("numberParam", 123)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"numberParam","params":[123],"id":1}`))

	rpcClient.Call("numberParams", 123, 321)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"numberParams","params":[123,321],"id":1}`))

	rpcClient.Call("floatParam", 1.23)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"floatParam","params":[1.23],"id":1}`))

	rpcClient.Call("floatParams", 1.23, 3.21)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"floatParams","params":[1.23,3.21],"id":1}`))

	rpcClient.Call("manyParams", "Alex", 35, true, nil, 2.34)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"manyParams","params":["Alex",35,true,null,2.34],"id":1}`))

	rpcClient.Call("emptyArray", []interface{}{})
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"emptyArray","params":[],"id":1}`))

	rpcClient.Call("emptyAnyArray", []string{})
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"emptyAnyArray","params":[],"id":1}`))

	rpcClient.Call("emptyObject", struct{}{})
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"emptyObject","params":{},"id":1}`))

	rpcClient.Call("emptyMissingPublicFieldObject", struct{ name string }{name: "Alex",})
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"emptyMissingPublicFieldObject","params":{},"id":1}`))

	rpcClient.Call("singleStruct", person)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"singleStruct","params":{"name":"Alex","age":35,"country":"Germany"},"id":1}`))

	rpcClient.Call("singlePointerToStruct", &person)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"singlePointerToStruct","params":{"name":"Alex","age":35,"country":"Germany"},"id":1}`))

	pp := &person
	rpcClient.Call("doublePointerStruct", &pp)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"doublePointerStruct","params":{"name":"Alex","age":35,"country":"Germany"},"id":1}`))

	rpcClient.Call("multipleStructs", person, &food)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"multipleStructs","params":[{"name":"Alex","age":35,"country":"Germany"},{"name":"Cuba Libre","ingredients":["rum","cola"]}],"id":1}`))

	rpcClient.Call("singleStructInArray", []interface{}{person})
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"singleStructInArray","params":[{"name":"Alex","age":35,"country":"Germany"}],"id":1}`))

	rpcClient.Call("namedParameters", map[string]interface{}{
		"name": "Alex",
		"age":  35,
	})
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"namedParameters","params":{"age":35,"name":"Alex"},"id":1}`))

	rpcClient.Call("anonymousStruct", struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{"Alex", 33})
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"anonymousStruct","params":{"name":"Alex","age":33},"id":1}`))

	rpcClient.Call("structWithNullField", struct {
		Name    string  `json:"name"`
		Address *string `json:"address"`
	}{"Alex", nil})
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"structWithNullField","params":{"name":"Alex","address":null},"id":1}`))
}

func TestRpcClient_CallFor(t *testing.T) {
	RegisterTestingT(t)
	rpcClient := NewClient(httpServer.URL)

	var i int
	responseBody = `{"jsonrpc":"2.0","result":3,"id":1}`
	err := rpcClient.CallFor(&i, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(i).To(Equal(3))

	i = 3
	responseBody = `{"jsonrpc":"2.0","result":null,"id":1}`
	err = rpcClient.CallFor(&i, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// i is not modified when result is empty since null (nil) value cannot be stored in int
	Expect(i).To(Equal(3))

	var pi *int
	responseBody = `{"jsonrpc":"2.0","result":4,"id":1}`
	err = rpcClient.CallFor(pi, "something", 1, 2, 3)
	<-requestChan
	Expect(err).NotTo(BeNil())
	Expect(pi).To(BeNil())

	responseBody = `{"jsonrpc":"2.0","result":4,"id":1}`
	err = rpcClient.CallFor(&pi, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(*pi).To(Equal(4))

	*pi = 3
	responseBody = `{"jsonrpc":"2.0","result":null,"id":1}`
	err = rpcClient.CallFor(&pi, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// since pi has a value it is not overwritten by null result
	Expect(pi).To(BeNil())

	p := &Person{}
	responseBody = `{"jsonrpc":"2.0","result":null,"id":1}`
	err = rpcClient.CallFor(p, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p).NotTo(BeNil())

	var p2 *Person
	responseBody = `{"jsonrpc":"2.0","result":null,"id":1}`
	err = rpcClient.CallFor(p2, "something", 1, 2, 3)
	<-requestChan
	Expect(err).NotTo(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p2).To(BeNil())

	p3 := Person{}
	responseBody = `{"jsonrpc":"2.0","result":null,"id":1}`
	err = rpcClient.CallFor(&p3, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p).NotTo(BeNil())

	p = &Person{Age: 35}
	responseBody = `{"jsonrpc":"2.0","result":{"name":"Alex"},"id":1}`
	err = rpcClient.CallFor(p, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p.Name).To(Equal("Alex"))
	Expect(p.Age).To(Equal(35))

	p2 = nil
	responseBody = `{"jsonrpc":"2.0","result":{"name":"Alex"},"id":1}`
	err = rpcClient.CallFor(p2, "something", 1, 2, 3)
	<-requestChan
	Expect(err).NotTo(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p2).To(BeNil())

	p2 = nil
	responseBody = `{"jsonrpc":"2.0","result":{"name":"Alex"},"id":1}`
	err = rpcClient.CallFor(&p2, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p2).NotTo(BeNil())
	Expect(p2.Name).To(Equal("Alex"))

	p3 = Person{Age: 35}
	responseBody = `{"jsonrpc":"2.0","result":{"name":"Alex"},"id":1}`
	err = rpcClient.CallFor(&p3, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p.Name).To(Equal("Alex"))
	Expect(p.Age).To(Equal(35))

	p3 = Person{Age: 35}
	responseBody = `{"jsonrpc":"2.0","result":{"name":"Alex"},"id":1}`
	err = rpcClient.CallFor(&p3, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p.Name).To(Equal("Alex"))
	Expect(p.Age).To(Equal(35))

	var intArray []int
	responseBody = `{"jsonrpc":"2.0","result":[1, 2, 3],"id":1}`
	err = rpcClient.CallFor(&intArray, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(intArray).To(ContainElement(1))
	Expect(intArray).To(ContainElement(2))
	Expect(intArray).To(ContainElement(3))
}

func TestRpcJsonResponseStruct(t *testing.T) {
	RegisterTestingT(t)
	rpcClient := NewClient(httpServer.URL)

	responseBody = ``
	response, err := rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	Expect(err).NotTo(BeNil())
	Expect(response).To(BeNil())

	responseBody = `{"result": null}`
	response, err = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	Expect(err).To(BeNil())
	Expect(response).NotTo(BeNil())
	Expect(response.Result).To(BeNil())


}

type Person struct {
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Country string `json:"country"`
}

type Drink struct {
	Name        string   `json:"name"`
	Ingredients []string `json:"ingredients"`
}

func TestReadmeExamples(t *testing.T) {
	RegisterTestingT(t)

	rpcClient := NewClient(httpServer.URL)

	rpcClient.Call("getDate")
	body := (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"getDate","id":1}`))

	rpcClient.Call("addNumbers", 1, 2)
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"addNumbers","params":[1,2],"id":1}`))

	rpcClient.Call("createPerson", "Alex", 33, "Germany")
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"createPerson","params":["Alex",33,"Germany"],"id":1}`))

	rpcClient.Call("createPerson", Person{"Alex", 33, "Germany"})
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"createPerson","params":{"name":"Alex","age":33,"country":"Germany"},"id":1}`))

	rpcClient.Call("createPersonsWithRole", []Person{{"Alex", 33, "Germany"}, {"Barney", 38, "Germany"}}, []string{"Admin", "User"})
	body = (<-requestChan).body
	Expect(body).To(Equal(`{"jsonrpc":"2.0","method":"createPersonsWithRole","params":[[{"name":"Alex","age":33,"country":"Germany"},{"name":"Barney","age":38,"country":"Germany"}],["Admin","User"]],"id":1}`))

}

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

// test if the structure of an rpc request is built correctly by validating the data that arrived on the test server
func TestRpcClient_Call(t *testing.T) {
	RegisterTestingT(t)
	rpcClient := NewClient(httpServer.URL)

	person := Person{
		Name:    "Alex",
		Age:     35,
		Country: "Germany",
	}

	drink := Drink{
		Name:        "Cuba Libre",
		Ingredients: []string{"rum", "cola"},
	}

	rpcClient.Call("missingParam")
	Expect((<-requestChan).body).To(Equal(`{"method":"missingParam","id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("nullParam", nil)
	Expect((<-requestChan).body).To(Equal(`{"method":"nullParam","params":[null],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("nullParams", nil, nil)
	Expect((<-requestChan).body).To(Equal(`{"method":"nullParams","params":[null,null],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("emptyParams", []interface{}{})
	Expect((<-requestChan).body).To(Equal(`{"method":"emptyParams","params":[],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("emptyAnyParams", []string{})
	Expect((<-requestChan).body).To(Equal(`{"method":"emptyAnyParams","params":[],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("emptyObject", struct{}{})
	Expect((<-requestChan).body).To(Equal(`{"method":"emptyObject","params":{},"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("emptyObjectList", []struct{}{{}, {}})
	Expect((<-requestChan).body).To(Equal(`{"method":"emptyObjectList","params":[{},{}],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("boolParam", true)
	Expect((<-requestChan).body).To(Equal(`{"method":"boolParam","params":[true],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("boolParams", true, false, true)
	Expect((<-requestChan).body).To(Equal(`{"method":"boolParams","params":[true,false,true],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("stringParam", "Alex")
	Expect((<-requestChan).body).To(Equal(`{"method":"stringParam","params":["Alex"],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("stringParams", "JSON", "RPC")
	Expect((<-requestChan).body).To(Equal(`{"method":"stringParams","params":["JSON","RPC"],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("numberParam", 123)
	Expect((<-requestChan).body).To(Equal(`{"method":"numberParam","params":[123],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("numberParams", 123, 321)
	Expect((<-requestChan).body).To(Equal(`{"method":"numberParams","params":[123,321],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("floatParam", 1.23)
	Expect((<-requestChan).body).To(Equal(`{"method":"floatParam","params":[1.23],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("floatParams", 1.23, 3.21)
	Expect((<-requestChan).body).To(Equal(`{"method":"floatParams","params":[1.23,3.21],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("manyParams", "Alex", 35, true, nil, 2.34)
	Expect((<-requestChan).body).To(Equal(`{"method":"manyParams","params":["Alex",35,true,null,2.34],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("emptyMissingPublicFieldObject", struct{ name string }{name: "Alex",})
	Expect((<-requestChan).body).To(Equal(`{"method":"emptyMissingPublicFieldObject","params":{},"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("singleStruct", person)
	Expect((<-requestChan).body).To(Equal(`{"method":"singleStruct","params":{"name":"Alex","age":35,"country":"Germany"},"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("singlePointerToStruct", &person)
	Expect((<-requestChan).body).To(Equal(`{"method":"singlePointerToStruct","params":{"name":"Alex","age":35,"country":"Germany"},"id":1,"jsonrpc":"2.0"}`))

	pp := &person
	rpcClient.Call("doublePointerStruct", &pp)
	Expect((<-requestChan).body).To(Equal(`{"method":"doublePointerStruct","params":{"name":"Alex","age":35,"country":"Germany"},"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("multipleStructs", person, &drink)
	Expect((<-requestChan).body).To(Equal(`{"method":"multipleStructs","params":[{"name":"Alex","age":35,"country":"Germany"},{"name":"Cuba Libre","ingredients":["rum","cola"]}],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("singleStructInArray", []interface{}{person})
	Expect((<-requestChan).body).To(Equal(`{"method":"singleStructInArray","params":[{"name":"Alex","age":35,"country":"Germany"}],"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("namedParameters", map[string]interface{}{
		"name": "Alex",
		"age":  35,
	})
	Expect((<-requestChan).body).To(Equal(`{"method":"namedParameters","params":{"age":35,"name":"Alex"},"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("anonymousStructNoTags", struct {
		Name string
		Age  int
	}{"Alex", 33})
	Expect((<-requestChan).body).To(Equal(`{"method":"anonymousStructNoTags","params":{"Name":"Alex","Age":33},"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("anonymousStructWithTags", struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{"Alex", 33})
	Expect((<-requestChan).body).To(Equal(`{"method":"anonymousStructWithTags","params":{"name":"Alex","age":33},"id":1,"jsonrpc":"2.0"}`))

	rpcClient.Call("structWithNullField", struct {
		Name    string  `json:"name"`
		Address *string `json:"address"`
	}{"Alex", nil})
	Expect((<-requestChan).body).To(Equal(`{"method":"structWithNullField","params":{"name":"Alex","address":null},"id":1,"jsonrpc":"2.0"}`))
}

// test if the result of an an rpc request is parsed correctly and if errors are thrown correctly
func TestRpcJsonResponseStruct(t *testing.T) {
	RegisterTestingT(t)
	rpcClient := NewClient(httpServer.URL)

	// empty return body is an error
	responseBody = ``
	res, err := rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).NotTo(BeNil())
	Expect(res).To(BeNil())

	// not a json body is an error
	responseBody = `{ "not": "a", "json": "object"`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).NotTo(BeNil())
	Expect(res).To(BeNil())

	// field "anotherField" not allowed in rpc response is an error
	responseBody = `{ "anotherField": "norpc"}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).NotTo(BeNil())
	Expect(res).To(BeNil())

	// TODO: result must contain one of "result", "error"
	// TODO: is there an efficient way to do this?
	/*responseBody = `{}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).NotTo(BeNil())
	Expect(res).To(BeNil())*/

	// result null is ok
	responseBody = `{"result": null}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Result).To(BeNil())
	Expect(res.Error).To(BeNil())

	// error null is ok
	responseBody = `{"error": null}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Result).To(BeNil())
	Expect(res.Error).To(BeNil())

	// result and error null is ok
	responseBody = `{"result": null, "error": null}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Result).To(BeNil())
	Expect(res.Error).To(BeNil())

	// TODO: result must not contain both of "result", "error" != null
	// TODO: is there an efficient way to do this?
	/*responseBody = `{ "result": 123, "error": {"code": 123, "message": "something wrong"}}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).NotTo(BeNil())
	Expect(res).To(BeNil())*/

	// result string is ok
	responseBody = `{"result": "ok"}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Result).To(Equal("ok"))

	// result with error null is ok
	responseBody = `{"result": "ok", "error": null}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Result).To(Equal("ok"))

	// error with result null is ok
	responseBody = `{"error": {"code": 123, "message": "something wrong"}, "result": null}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Result).To(BeNil())
	Expect(res.Error.Code).To(Equal(123))
	Expect(res.Error.Message).To(Equal("something wrong"))

	// TODO: empty error is not ok, must at least contain code and message
	/*responseBody = `{ "error": {}}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Result).To(BeNil())
	Expect(res.Error).NotTo(BeNil())*/

	// TODO: only code in error is not ok, must at least contain code and message
	/*responseBody = `{ "error": {"code": 123}}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Result).To(BeNil())
	Expect(res.Error).NotTo(BeNil())*/

	// TODO: only message in error is not ok, must at least contain code and message
	/*responseBody = `{ "error": {"message": "something wrong"}}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Result).To(BeNil())
	Expect(res.Error).NotTo(BeNil())*/

	// error with code and message is ok
	responseBody = `{ "error": {"code": 123, "message": "something wrong"}}`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Result).To(BeNil())
	Expect(res.Error.Code).To(Equal(123))
	Expect(res.Error.Message).To(Equal("something wrong"))

	// check results

	// should return int correctly
	responseBody = `{ "result": 1 }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	i, err := res.GetInt()
	Expect(err).To(BeNil())
	Expect(i).To(Equal(int64(1)))

	// error on wrong type
	i = 3
	responseBody = `{ "result": "notAnInt" }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	i, err = res.GetInt()
	Expect(err).NotTo(BeNil())
	Expect(i).To(Equal(int64(0)))

	// error on result null
	i = 3
	responseBody = `{ "result": null }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	i, err = res.GetInt()
	Expect(err).NotTo(BeNil())
	Expect(i).To(Equal(int64(0)))

	b := false
	responseBody = `{ "result": true }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	b, err = res.GetBool()
	Expect(err).To(BeNil())
	Expect(b).To(Equal(true))

	b = true
	responseBody = `{ "result": 123 }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	b, err = res.GetBool()
	Expect(err).NotTo(BeNil())
	Expect(b).To(Equal(false))

	var p *Person
	responseBody = `{ "result": {"name": "Alex", "age": 35, "anotherField": "something"} }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	err = res.GetObject(&p)
	Expect(err).To(BeNil())
	Expect(p.Name).To(Equal("Alex"))
	Expect(p.Age).To(Equal(35))
	Expect(p.Country).To(Equal(""))

	// TODO: How to check if result could be parsed or if it is default?
	p = nil
	responseBody = `{ "result": {"anotherField": "something"} }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	err = res.GetObject(&p)
	Expect(err).To(BeNil())
	Expect(p).NotTo(BeNil())

	// TODO: HERE######
	var pp *PointerFieldPerson
	responseBody = `{ "result": {"anotherField": "something", "country": "Germany"} }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	err = res.GetObject(&pp)
	Expect(err).To(BeNil())
	Expect(pp.Name).To(BeNil())
	Expect(pp.Age).To(BeNil())
	Expect(*pp.Country).To(Equal("Germany"))

	p = nil
	responseBody = `{ "result": null }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	err = res.GetObject(&p)
	Expect(err).To(BeNil())
	Expect(p).To(BeNil())

	// passing nil is an error
	p = nil
	responseBody = `{ "result": null }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	err = res.GetObject(p)
	Expect(err).NotTo(BeNil())
	Expect(p).To(BeNil())

	p2 := &Person{
		Name: "Alex",
	}
	responseBody = `{ "result": null }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	err = res.GetObject(&p2)
	Expect(err).To(BeNil())
	Expect(p2).To(BeNil())

	p2 = &Person{
		Name: "Alex",
	}
	responseBody = `{ "result": {"age": 35} }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	err = res.GetObject(p2)
	Expect(err).To(BeNil())
	Expect(p2.Name).To(Equal("Alex"))
	Expect(p2.Age).To(Equal(35))

	// prefilled struct is kept on no result
	p3 := Person{
		Name: "Alex",
	}
	responseBody = `{ "result": null }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	err = res.GetObject(&p3)
	Expect(err).To(BeNil())
	Expect(p3.Name).To(Equal("Alex"))

	// prefilled struct is extended / overwritten
	p3 = Person{
		Name: "Alex",
		Age:  123,
	}
	responseBody = `{ "result": {"age": 35, "country": "Germany"} }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	err = res.GetObject(&p3)
	Expect(err).To(BeNil())
	Expect(p3.Name).To(Equal("Alex"))
	Expect(p3.Age).To(Equal(35))
	Expect(p3.Country).To(Equal("Germany"))

	// nil is an error
	responseBody = `{ "result": {"age": 35} }`
	res, err = rpcClient.Call("something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(res.Error).To(BeNil())
	err = res.GetObject(nil)
	Expect(err).NotTo(BeNil())
}

func TestRpcClient_CallFor(t *testing.T) {
	RegisterTestingT(t)
	rpcClient := NewClient(httpServer.URL)

	i := 0
	responseBody = `{"result":3,"id":1,"jsonrpc":"2.0"}`
	err := rpcClient.CallFor(&i, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(i).To(Equal(3))

	/*
	i = 3
	responseBody = `{"result":null,"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(&i, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// i is not modified when result is empty since null (nil) value cannot be stored in int
	Expect(i).To(Equal(3))

	var pi *int
	responseBody = `{"result":4,"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(pi, "something", 1, 2, 3)
	<-requestChan
	Expect(err).NotTo(BeNil())
	Expect(pi).To(BeNil())

	responseBody = `{"result":4,"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(&pi, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	Expect(*pi).To(Equal(4))

	*pi = 3
	responseBody = `{"result":null,"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(&pi, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// since pi has a value it is not overwritten by null result
	Expect(pi).To(BeNil())

	p := &Person{}
	responseBody = `{"result":null,"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(p, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p).NotTo(BeNil())

	var p2 *Person
	responseBody = `{"result":null,"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(p2, "something", 1, 2, 3)
	<-requestChan
	Expect(err).NotTo(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p2).To(BeNil())

	p3 := Person{}
	responseBody = `{"result":null,"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(&p3, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p).NotTo(BeNil())

	p = &Person{Age: 35}
	responseBody = `{"result":{"name":"Alex"},"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(p, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p.Name).To(Equal("Alex"))
	Expect(p.Age).To(Equal(35))

	p2 = nil
	responseBody = `{"result":{"name":"Alex"},"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(p2, "something", 1, 2, 3)
	<-requestChan
	Expect(err).NotTo(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p2).To(BeNil())

	p2 = nil
	responseBody = `{"result":{"name":"Alex"},"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(&p2, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p2).NotTo(BeNil())
	Expect(p2.Name).To(Equal("Alex"))

	p3 = Person{Age: 35}
	responseBody = `{"result":{"name":"Alex"},"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(&p3, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p.Name).To(Equal("Alex"))
	Expect(p.Age).To(Equal(35))

	p3 = Person{Age: 35}
	responseBody = `{"result":{"name":"Alex"},"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(&p3, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(p.Name).To(Equal("Alex"))
	Expect(p.Age).To(Equal(35))

	var intArray []int
	responseBody = `{"result":[1, 2, 3],"id":1,"jsonrpc":"2.0"}`
	err = rpcClient.CallFor(&intArray, "something", 1, 2, 3)
	<-requestChan
	Expect(err).To(BeNil())
	// p is not changed since it has a value and result is null
	Expect(intArray).To(ContainElement(1))
	Expect(intArray).To(ContainElement(2))
	Expect(intArray).To(ContainElement(3))*/
}

type Person struct {
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Country string `json:"country"`
}

type PointerFieldPerson struct {
	Name    *string `json:"name"`
	Age     *int    `json:"age"`
	Country *string `json:"country"`
}

type Drink struct {
	Name        string   `json:"name"`
	Ingredients []string `json:"ingredients"`
}

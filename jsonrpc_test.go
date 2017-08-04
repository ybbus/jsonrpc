package jsonrpc

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"time"

	"github.com/onsi/gomega"
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
	gomega.RegisterTestingT(t)

	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.Call("add", 1, 2)

	req := (<-requestChan).request

	gomega.Expect(req.Method).To(gomega.Equal("POST"))
	gomega.Expect(req.Header.Get("Content-Type")).To(gomega.Equal("application/json"))
	gomega.Expect(req.Header.Get("Accept")).To(gomega.Equal("application/json"))
}

// test if the structure of an rpc request is built correctly validate the data that arrived on the server
func TestRpcJsonRequestStruct(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.SetAutoIncrementID(false)

	rpcClient.Call("add", 1, 2)
	body := (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"add","params":[1,2],"id":0}`))

	rpcClient.Call("setName", "alex")
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"setName","params":["alex"],"id":0}`))

	rpcClient.Call("setPerson", "alex", 33, "Germany")
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"setPerson","params":["alex",33,"Germany"],"id":0}`))

	rpcClient.Call("setPersonObject", Person{"alex", 33, "Germany"})
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"setPersonObject","params":[{"name":"alex","age":33,"country":"Germany"}],"id":0}`))

	rpcClient.Call("getDate")
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"getDate","id":0}`))

	rpcClient.Call("setAnonymStruct", struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{"Alex", 33})
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"setAnonymStruct","params":[{"name":"Alex","age":33}],"id":0}`))
}

// test if the structure of an rpc request is built correctly validate the data that arrived on the server
func TestRpcJsonRequestStructWithNamedParams(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.SetAutoIncrementID(false)

	rpcClient.CallNamed("myMethod", map[string]interface{}{
		"arrayOfInts":    []int{1, 2, 3},
		"arrayOfStrings": []string{"A", "B", "C"},
		"bool":           true,
		"int":            1,
		"number":         1.2,
		"string":         "boogaloo",
		"subObject":      map[string]interface{}{"foo": "bar"},
	})
	body := (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"myMethod","params":{"arrayOfInts":[1,2,3],"arrayOfStrings":["A","B","C"],"bool":true,"int":1,"number":1.2,"string":"boogaloo","subObject":{"foo":"bar"}},"id":0}`))
}
func TestRpcJsonResponseStruct(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.SetAutoIncrementID(false)

	responseBody = `{"jsonrpc":"2.0","result":3,"id":0}`
	response, _ := rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var int64Result int64
	int64Result, _ = response.GetInt64()
	gomega.Expect(int64Result).To(gomega.Equal(int64(3)))

	responseBody = `{"jsonrpc":"2.0","result":3,"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var intResult int
	intResult, _ = response.GetInt()
	gomega.Expect(intResult).To(gomega.Equal(3))

	responseBody = `{"jsonrpc":"2.0","result":3.3,"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	_, err := response.GetInt()
	gomega.Expect(err).To(gomega.Not(gomega.Equal(nil)))

	responseBody = `{"jsonrpc":"2.0","result":false,"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	_, err = response.GetInt()
	gomega.Expect(err).To(gomega.Not(gomega.Equal(nil)))

	responseBody = `{"jsonrpc":"2.0","result": 3.7,"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var float64Result float64
	float64Result, _ = response.GetFloat64()
	gomega.Expect(float64Result).To(gomega.Equal(3.7))

	responseBody = `{"jsonrpc":"2.0","result": "1.3","id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	_, err = response.GetFloat64()
	gomega.Expect(err).To(gomega.Not(gomega.Equal(nil)))

	responseBody = `{"jsonrpc":"2.0","result": true,"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var boolResult bool
	boolResult, _ = response.GetBool()
	gomega.Expect(boolResult).To(gomega.Equal(true))

	responseBody = `{"jsonrpc":"2.0","result": 0,"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	_, err = response.GetBool()
	gomega.Expect(err).To(gomega.Not(gomega.Equal(nil)))

	responseBody = `{"jsonrpc":"2.0","result": "alex","id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var stringResult string
	stringResult, _ = response.GetString()
	gomega.Expect(stringResult).To(gomega.Equal("alex"))

	responseBody = `{"jsonrpc":"2.0","result": 123,"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	_, err = response.GetString()
	gomega.Expect(err).To(gomega.Not(gomega.Equal(nil)))

	responseBody = `{"jsonrpc":"2.0","result": {"name": "alex", "age": 33, "country": "Germany"},"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var person Person
	response.GetObject(&person)
	gomega.Expect(person).To(gomega.Equal(Person{"alex", 33, "Germany"}))

	responseBody = `{"jsonrpc":"2.0","result": 3,"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var number int
	response.GetObject(&number)
	gomega.Expect(int(number)).To(gomega.Equal(3))

	responseBody = `{"jsonrpc":"2.0","result": [{"name": "alex", "age": 33, "country": "Germany"}, {"name": "Ferolaz", "age": 333, "country": "Azeroth"}],"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var personArray = []Person{}
	response.GetObject(&personArray)
	gomega.Expect(personArray).To(gomega.Equal([]Person{{"alex", 33, "Germany"}, {"Ferolaz", 333, "Azeroth"}}))

	responseBody = `{"jsonrpc":"2.0","result": [1, 2, 3],"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var intArray []int
	response.GetObject(&intArray)
	gomega.Expect(intArray).To(gomega.Equal([]int{1, 2, 3}))
}

func TestResponseErrorWorks(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.SetAutoIncrementID(false)

	responseBody = `{"jsonrpc":"2.0","error": {"code": -123, "message": "something wrong"},"id":0}`
	response, _ := rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	gomega.Expect(*response.Error).To(gomega.Equal(RPCError{-123, "something wrong", nil}))
}

func TestNotifyWorks(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)

	rpcClient.Notification("test", 10)
	<-requestChan
	rpcClient.Notification("test", Person{"alex", 33, "Germany"})
	<-requestChan
	rpcClient.Notification("test", 10, 20, "alex")
	body := (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"test","params":[10,20,"alex"]}`))
}

func TestNewRPCRequestObject(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)

	req := rpcClient.NewRPCRequestObject("add", 1, 2)
	gomega.Expect(req).To(gomega.Equal(&RPCRequest{
		JSONRPC: "2.0",
		ID:      0,
		Method:  "add",
		Params:  []interface{}{1, 2},
	}))

	req = rpcClient.NewRPCRequestObject("getDate")
	gomega.Expect(req).To(gomega.Equal(&RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "getDate",
		Params:  nil,
	}))

	req = rpcClient.NewRPCRequestObject("getPerson", Person{"alex", 33, "germany"})
	gomega.Expect(req).To(gomega.Equal(&RPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "getPerson",
		Params:  []interface{}{Person{"alex", 33, "germany"}},
	}))
}

func TestNewRPCNotificationObject(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)

	req := rpcClient.NewRPCNotificationObject("add", 1, 2)
	gomega.Expect(req).To(gomega.Equal(&RPCNotification{
		JSONRPC: "2.0",
		Method:  "add",
		Params:  []interface{}{1, 2},
	}))

	req = rpcClient.NewRPCNotificationObject("getDate")
	gomega.Expect(req).To(gomega.Equal(&RPCNotification{
		JSONRPC: "2.0",
		Method:  "getDate",
		Params:  nil,
	}))

	req = rpcClient.NewRPCNotificationObject("getPerson", Person{"alex", 33, "germany"})
	gomega.Expect(req).To(gomega.Equal(&RPCNotification{
		JSONRPC: "2.0",
		Method:  "getPerson",
		Params:  []interface{}{Person{"alex", 33, "germany"}},
	}))
}

func TestBatchRequestWorks(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.SetCustomHeader("Test", "test")

	req1 := rpcClient.NewRPCRequestObject("test1", "alex")
	rpcClient.Batch(req1)
	req := <-requestChan
	body := req.body
	gomega.Expect(req.request.Header.Get("Test")).To(gomega.Equal("test"))
	gomega.Expect(body).To(gomega.Equal(`[{"jsonrpc":"2.0","method":"test1","params":["alex"],"id":0}]`))

	notify1 := rpcClient.NewRPCNotificationObject("test2", "alex")
	rpcClient.Batch(notify1)
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`[{"jsonrpc":"2.0","method":"test2","params":["alex"]}]`))

	rpcClient.Batch(req1, notify1)
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`[{"jsonrpc":"2.0","method":"test1","params":["alex"],"id":0},{"jsonrpc":"2.0","method":"test2","params":["alex"]}]`))

	requests := []interface{}{req1, notify1}
	rpcClient.Batch(requests...)
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`[{"jsonrpc":"2.0","method":"test1","params":["alex"],"id":0},{"jsonrpc":"2.0","method":"test2","params":["alex"]}]`))

	invalid := &Person{"alex", 33, "germany"}
	_, err := rpcClient.Batch(invalid, notify1)
	gomega.Expect(err).To(gomega.Not(gomega.Equal(nil)))
}

func TestBatchResponseWorks(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)

	responseBody = `[{"jsonrpc":"2.0","result": 1,"id":0},{"jsonrpc":"2.0","result": 2,"id":1},{"jsonrpc":"2.0","result": 3,"id":3}]`
	req1 := rpcClient.NewRPCRequestObject("test1", 1)
	req2 := rpcClient.NewRPCRequestObject("test2", 2)
	req3 := rpcClient.NewRPCRequestObject("test3", 3)
	responses, _ := rpcClient.Batch(req1, req2, req3)
	<-requestChan

	resp2, _ := responses.GetResponseOf(req2)
	res2, _ := resp2.GetInt()

	gomega.Expect(res2).To(gomega.Equal(2))
}

func TestIDIncremtWorks(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.SetAutoIncrementID(true) // default

	rpcClient.Call("test1", 1, 2)
	body := (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"test1","params":[1,2],"id":0}`))

	rpcClient.Call("test2", 1, 2)
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"test2","params":[1,2],"id":1}`))

	rpcClient.SetNextID(10)

	rpcClient.Call("test3", 1, 2)
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"test3","params":[1,2],"id":10}`))

	rpcClient.Call("test4", 1, 2)
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"test4","params":[1,2],"id":11}`))

	rpcClient.SetAutoIncrementID(false)

	rpcClient.Call("test5", 1, 2)
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"test5","params":[1,2],"id":12}`))

	rpcClient.Call("test6", 1, 2)
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"test6","params":[1,2],"id":12}`))
}

func TestRequestIDUpdateWorks(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.SetAutoIncrementID(true) // default

	req1 := rpcClient.NewRPCRequestObject("test", 1, 2, 3)
	req2 := rpcClient.NewRPCRequestObject("test", 1, 2, 3)
	gomega.Expect(int(req1.ID)).To(gomega.Equal(0))
	gomega.Expect(int(req2.ID)).To(gomega.Equal(1))

	rpcClient.UpdateRequestID(req1)
	rpcClient.UpdateRequestID(req2)

	gomega.Expect(int(req1.ID)).To(gomega.Equal(2))
	gomega.Expect(int(req2.ID)).To(gomega.Equal(3))

	rpcClient.UpdateRequestID(req2)
	rpcClient.UpdateRequestID(req1)

	gomega.Expect(int(req1.ID)).To(gomega.Equal(5))
	gomega.Expect(int(req2.ID)).To(gomega.Equal(4))

	rpcClient.UpdateRequestID(req1)
	rpcClient.UpdateRequestID(req1)

	gomega.Expect(int(req1.ID)).To(gomega.Equal(7))
	gomega.Expect(int(req2.ID)).To(gomega.Equal(4))

	rpcClient.SetAutoIncrementID(false)

	rpcClient.UpdateRequestID(req2)
	rpcClient.UpdateRequestID(req1)

	gomega.Expect(int(req1.ID)).To(gomega.Equal(8))
	gomega.Expect(int(req2.ID)).To(gomega.Equal(8))

	rpcClient.SetAutoIncrementID(false)

}

func TestBasicAuthentication(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)

	rpcClient.SetBasicAuth("alex", "secret")
	rpcClient.Call("add", 1, 2)
	req := (<-requestChan).request
	gomega.Expect(req.Header.Get("Authorization")).To(gomega.Equal("Basic YWxleDpzZWNyZXQ="))

	rpcClient.SetBasicAuth("", "")
	rpcClient.Call("add", 1, 2)
	req = (<-requestChan).request
	gomega.Expect(req.Header.Get("Authorization")).NotTo(gomega.Equal("Basic YWxleDpzZWNyZXQ="))
}

func TestCustomHeaders(t *testing.T) {
	gomega.RegisterTestingT(t)

	rpcClient := NewRPCClient(httpServer.URL)

	rpcClient.SetCustomHeader("Test", "success")
	rpcClient.Call("add", 1, 2)
	req := (<-requestChan).request

	gomega.Expect(req.Header.Get("Test")).To(gomega.Equal("success"))

	rpcClient.SetCustomHeader("Test2", "success2")
	rpcClient.Call("add", 1, 2)
	req = (<-requestChan).request

	gomega.Expect(req.Header.Get("Test")).To(gomega.Equal("success"))
	gomega.Expect(req.Header.Get("Test2")).To(gomega.Equal("success2"))

	rpcClient.UnsetCustomHeader("Test")
	rpcClient.Call("add", 1, 2)
	req = (<-requestChan).request

	gomega.Expect(req.Header.Get("Test")).NotTo(gomega.Equal("success"))
	gomega.Expect(req.Header.Get("Test2")).To(gomega.Equal("success2"))
}

func TestCustomHTTPClient(t *testing.T) {
	gomega.RegisterTestingT(t)

	rpcClient := NewRPCClient(httpServer.URL)

	proxyURL, _ := url.Parse("http://proxy:8080")
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}

	httpClient := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	rpcClient.SetHTTPClient(httpClient)
	rpcClient.Call("add", 1, 2)
	// req := (<-requestChan).request
	// TODO: what to test here?
}

type Person struct {
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Country string `json:"country"`
}

func TestReadmeExamples(t *testing.T) {
	gomega.RegisterTestingT(t)

	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.SetAutoIncrementID(false)

	rpcClient.Call("getDate")
	body := (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"getDate","id":0}`))

	rpcClient.Call("addNumbers", 1, 2)
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"addNumbers","params":[1,2],"id":0}`))

	rpcClient.Call("createPerson", "Alex", 33, "Germany")
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"createPerson","params":["Alex",33,"Germany"],"id":0}`))

	rpcClient.Call("createPerson", Person{"Alex", 33, "Germany"})
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"createPerson","params":[{"name":"Alex","age":33,"country":"Germany"}],"id":0}`))

	rpcClient.Call("createPersonsWithRole", []Person{{"Alex", 33, "Germany"}, {"Barney", 38, "Germany"}}, []string{"Admin", "User"})
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`{"jsonrpc":"2.0","method":"createPersonsWithRole","params":[[{"name":"Alex","age":33,"country":"Germany"},{"name":"Barney","age":38,"country":"Germany"}],["Admin","User"]],"id":0}`))

}

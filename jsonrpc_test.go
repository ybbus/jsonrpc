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

var requestChan = make(chan *RequestData, 1)

var responseBody = ""

type RequestData struct {
	request *http.Request
	body    string
}

var httpServer *httptest.Server

func TestMain(m *testing.M) {
	httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
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

func TestRpcJsonRequestStruct(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.SetAutoIncrementID(false)

	testData := []struct {
		inMethod      string
		inParams      []interface{}
		outResultJSON string
	}{
		{"add", []interface{}{1, 2}, `{"jsonrpc":"2.0","method":"add","params":[1,2],"id":0}`},
		{"setName", []interface{}{"alex"}, `{"jsonrpc":"2.0","method":"setName","params":["alex"],"id":0}`},
		{"setPerson", []interface{}{"alex", 33, "Germany"}, `{"jsonrpc":"2.0","method":"setPerson","params":["alex",33,"Germany"],"id":0}`},
		{"getDate", []interface{}{}, `{"jsonrpc":"2.0","method":"getDate","id":0}`},
		{"setObject", []interface{}{struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}{"Alex", 33}}, `{"jsonrpc":"2.0","method":"setObject","params":[{"name":"Alex","age":33}],"id":0}`},
	}

	for _, test := range testData {
		rpcClient.Call(test.inMethod, test.inParams...)
		body := (<-requestChan).body
		gomega.Expect(body).To(gomega.Equal(test.outResultJSON))
	}
}

func TestRpcJsonResponseStruct(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.SetAutoIncrementID(false)

	responseBody = `{"jsonrpc":"2.0","result":3,"id":0}`
	response, _ := rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var intResult int64
	intResult, _ = response.GetInt()
	gomega.Expect(int(intResult)).To(gomega.Equal(3))

	responseBody = `{"jsonrpc":"2.0","result": 3.7,"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var floatResult float64
	floatResult, _ = response.GetFloat()
	gomega.Expect(floatResult).To(gomega.Equal(3.7))

	responseBody = `{"jsonrpc":"2.0","result": true,"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var boolResult bool
	boolResult, _ = response.GetBool()
	gomega.Expect(boolResult).To(gomega.Equal(true))

	responseBody = `{"jsonrpc":"2.0","result": {"name": "alex", "age": 33, "country": "Germany"},"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var person = Person{}
	response.GetObject(&person)
	gomega.Expect(person).To(gomega.Equal(Person{"alex", 33, "Germany"}))

	responseBody = `{"jsonrpc":"2.0","result": [{"name": "alex", "age": 33, "country": "Germany"}, {"name": "Ferolaz", "age": 333, "country": "Azeroth"}],"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var personArray = []Person{}
	response.GetObject(&personArray)
	gomega.Expect(personArray).To(gomega.Equal([]Person{{"alex", 33, "Germany"}, {"Ferolaz", 333, "Azeroth"}}))

	responseBody = `{"jsonrpc":"2.0","result": [1, 2, 3],"id":0}`
	response, _ = rpcClient.Call("test") // Call param does not matter, since response does not depend on request
	<-requestChan
	var intArray = []int{}
	response.GetObject(&intArray)
	gomega.Expect(intArray).To(gomega.Equal([]int{1, 2, 3}))
}

func TestResponseErrorWorks(t *testing.T) {
	// TODO
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

func TestBatchRequestWorks(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)

	req1 := rpcClient.NewRPCRequestObject("test1", "alex")
	rpcClient.Batch(req1)
	body := (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`[{"jsonrpc":"2.0","method":"test1","params":["alex"],"id":0}]`))

	notify1 := rpcClient.NewRPCNotificationObject("test2", "alex")
	rpcClient.Batch(notify1)
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`[{"jsonrpc":"2.0","method":"test2","params":["alex"]}]`))

	rpcClient.Batch(req1, notify1)
	body = (<-requestChan).body
	gomega.Expect(body).To(gomega.Equal(`[{"jsonrpc":"2.0","method":"test1","params":["alex"],"id":0},{"jsonrpc":"2.0","method":"test2","params":["alex"]}]`))
}

func TestBatchResponseWorks(t *testing.T) {
	gomega.RegisterTestingT(t)
	rpcClient := NewRPCClient(httpServer.URL)

	responseBody = `[{"jsonrpc":"2.0","result": {"name": "alex", "age": 33, "country": "Germany"},"id":0},{"jsonrpc":"2.0","result": 42,"id":1}]`
	req1 := rpcClient.NewRPCRequestObject("test1", "alex")
	response, _ := rpcClient.Batch(req1)
	<-requestChan
	p := Person{}
	response[0].GetObject(&p)
	resp2, _ := response[1].GetInt()
	gomega.Expect(p).To(gomega.Equal(Person{"alex", 33, "Germany"}))
	gomega.Expect(int(resp2)).To(gomega.Equal(42))
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

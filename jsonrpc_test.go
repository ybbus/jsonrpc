package jsonrpc

import (
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

package jsonrpc

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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

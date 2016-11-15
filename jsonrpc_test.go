package jsonrpc

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/onsi/gomega"
)

var requestChan = make(chan *http.Request, 1)

var httpServer *httptest.Server

func TestMain(m *testing.M) {
	httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requestChan <- r
	}))
	defer httpServer.Close()

	os.Exit(m.Run())
}

func TestSimpleRpcCallHeaderCorrect(t *testing.T) {
	gomega.RegisterTestingT(t)

	rpcClient := NewRPCClient(httpServer.URL)
	rpcClient.Call("add", 1, 2)

	req, _ := getLatestRequest()

	gomega.Expect(req.Method).To(gomega.Equal("POST"))
	gomega.Expect(req.Header.Get("Content-Type")).To(gomega.Equal("application/json"))
	gomega.Expect(req.Header.Get("Accept")).To(gomega.Equal("application/json"))
}

func getLatestRequest() (*http.Request, string) {
	req := <-requestChan
	body, _ := ioutil.ReadAll(req.Body)
	req.Body.Close()
	return req, string(body)
}

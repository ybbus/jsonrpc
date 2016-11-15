package jsonrpc

import "net/http"
import "encoding/json"
import "bytes"
import "io/ioutil"

// RPCClient returns client that is used to execute json rpc calls over http
type RPCClient interface {
	Call(string, ...interface{}) (RPCResponse, error)
}

type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Error   RPCError    `json:"error"`
	ID      int         `json:"id"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type rpcClient struct {
	endpoint   string
	httpClient *http.Client
}

func NewRPCClient(endpoint string) RPCClient {
	return &rpcClient{
		endpoint:   endpoint,
		httpClient: http.DefaultClient,
	}
}

func (client *rpcClient) Call(method string, params ...interface{}) (RPCResponse, error) {
	httpRequest := client.newRequest(method, params...)
	rpcResponse := RPCResponse{}

	httpResponse, _ := client.httpClient.Do(httpRequest)
	defer httpResponse.Body.Close()

	body, _ := ioutil.ReadAll(httpResponse.Body)

	json.Unmarshal(body, &rpcResponse)

	return rpcResponse, nil
}

func (client *rpcClient) newRequest(method string, params ...interface{}) *http.Request {
	rpcRequest := RPCRequest{
		ID:      0,
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	body, _ := json.Marshal(&rpcRequest)
	request, _ := http.NewRequest("POST", client.endpoint, bytes.NewReader(body))
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	return request
}

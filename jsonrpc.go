package jsonrpc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
)

// RPCClient returns client that is used to execute json rpc calls over http
type RPCClient interface {
	Call(string, ...interface{}) (RPCResponse, error)
	SetNextID(uint)
	SetAutoIncrementID(bool)
	SetBasicAuth(string, string)
	SetHTTPClient(*http.Client)
}

// RPCRequest is the structure that is used to build up an json-rpc request.
// See: http://www.jsonrpc.org/specification#request_object
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      uint        `json:"id"`
}

// RPCResponse is the structure that is used to provide the result of an json-rpc request.
// See: http://www.jsonrpc.org/specification#response_object
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Error   RPCError    `json:"error"`
	ID      int         `json:"id"`
}

// RPCError is the structure that is used to provide the result in case of an rpc call error.
// See: http://www.jsonrpc.org/specification#error_object
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type rpcClient struct {
	endpoint        string
	httpClient      *http.Client
	basicAuth       string
	autoIncrementID bool
	nextID          uint
	idMutex         sync.Mutex
}

// NewRPCClient returns a new RPCClient interface with default configuration
func NewRPCClient(endpoint string) RPCClient {
	return &rpcClient{
		endpoint:        endpoint,
		httpClient:      http.DefaultClient,
		autoIncrementID: true,
		nextID:          0,
	}
}

func (client *rpcClient) Call(method string, params ...interface{}) (RPCResponse, error) {
	rpcResponse := RPCResponse{}
	httpRequest, err := client.newRequest(method, params...)
	if err != nil {
		return rpcResponse, err
	}

	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return rpcResponse, err
	}
	defer httpResponse.Body.Close()

	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return rpcResponse, err
	}

	err = json.Unmarshal(body, &rpcResponse)
	if err != nil {
		return rpcResponse, err
	}

	return rpcResponse, nil
}

func (client *rpcClient) SetAutoIncrementID(flag bool) {
	client.autoIncrementID = flag
}

func (client *rpcClient) SetNextID(id uint) {
	client.idMutex.Lock()
	client.nextID = id
	client.idMutex.Unlock()
}

func (client *rpcClient) incrementID() {
	client.idMutex.Lock()
	client.nextID++
	client.idMutex.Unlock()
}

func (client *rpcClient) SetBasicAuth(username string, password string) {
	auth := username + ":" + password
	client.basicAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func (client *rpcClient) SetHTTPClient(httpClient *http.Client) {
	client.httpClient = httpClient
}

func (client *rpcClient) newRequest(method string, params ...interface{}) (*http.Request, error) {
	client.idMutex.Lock()
	rpcRequest := RPCRequest{
		ID:      client.nextID,
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	if client.autoIncrementID == true {
		client.nextID++
	}
	client.idMutex.Unlock()

	if len(params) == 0 {
		rpcRequest.Params = nil
	}

	body, err := json.Marshal(&rpcRequest)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", client.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if client.basicAuth != "" {
		request.Header.Add("Authorization", client.basicAuth)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")

	return request, nil
}

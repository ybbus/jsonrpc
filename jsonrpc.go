package jsonrpc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// RPCClient returns client that is used to execute json rpc calls over http
type RPCClient interface {
	Call(method string, params ...interface{}) (*RPCResponse, error)
	Notify(method string, params ...interface{}) error
	SetNextID(id uint)
	SetAutoIncrementID(flag bool)
	SetBasicAuth(username string, password string)
	SetHTTPClient(httpClient *http.Client)
	SetCustomHeader(key string, value string)
}

// RPCRequest is the structure that is used to build up an json-rpc request.
// See: http://www.jsonrpc.org/specification#request_object
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      uint        `json:"id"`
}

type RPCNotify struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// RPCResponse is the structure that is used to provide the result of an json-rpc request.
// See: http://www.jsonrpc.org/specification#response_object
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
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
	customHeaders   map[string]string
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
		customHeaders:   make(map[string]string),
	}
}

func (client *rpcClient) Call(method string, params ...interface{}) (*RPCResponse, error) {
	httpRequest, err := client.newRequest(false, method, params...)
	if err != nil {
		return nil, err
	}

	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	rpcResponse := RPCResponse{}
	decoder := json.NewDecoder(httpResponse.Body)
	decoder.UseNumber()
	err = decoder.Decode(&rpcResponse)
	if err != nil {
		return nil, err
	}

	return &rpcResponse, nil
}

func (client *rpcClient) Notify(method string, params ...interface{}) error {
	httpRequest, err := client.newRequest(true, method, params...)
	if err != nil {
		return err
	}

	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	return nil
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

func (client *rpcClient) SetCustomHeader(key string, value string) {
	client.customHeaders[key] = value
}

func (client *rpcClient) SetBasicAuth(username string, password string) {
	auth := username + ":" + password
	client.basicAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func (client *rpcClient) SetHTTPClient(httpClient *http.Client) {
	client.httpClient = httpClient
}

func (client *rpcClient) newRequest(notification bool, method string, params ...interface{}) (*http.Request, error) {

	// TODO: easier way to remote ID from RPCRequest without extra struct
	var rpcRequest interface{}
	if notification {
		notify := RPCNotify{
			JSONRPC: "2.0",
			Method:  method,
			Params:  params,
		}
		if len(params) == 0 {
			notify.Params = nil
		}
		rpcRequest = notify
	} else {
		client.idMutex.Lock()
		request := RPCRequest{
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
			request.Params = nil
		}
		rpcRequest = request
	}

	body, err := json.Marshal(rpcRequest)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", client.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for k, v := range client.customHeaders {
		request.Header.Add(k, v)
	}

	if client.basicAuth != "" {
		request.Header.Add("Authorization", client.basicAuth)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")

	return request, nil
}

func (rpcResponse *RPCResponse) GetInt() (int64, error) {
	val, ok := rpcResponse.Result.(json.Number)
	if !ok {
		return 0, fmt.Errorf("could not parse int from %s", rpcResponse.Result)
	}

	i, err := val.Int64()
	if err != nil {
		return 0, err
	}

	return i, nil
}

func (rpcResponse *RPCResponse) GetFloat() (float64, error) {
	val, ok := rpcResponse.Result.(json.Number)
	if !ok {
		return 0, fmt.Errorf("could not parse int from %s", rpcResponse.Result)
	}

	f, err := val.Float64()
	if err != nil {
		return 0, err
	}

	return f, nil
}

func (rpcResponse *RPCResponse) GetBool() (bool, error) {
	val, ok := rpcResponse.Result.(bool)
	if !ok {
		return false, fmt.Errorf("could not parse int from %s", rpcResponse.Result)
	}

	return val, nil
}

func (rpcResponse *RPCResponse) GetString() (string, error) {
	val, ok := rpcResponse.Result.(string)
	if !ok {
		return "", fmt.Errorf("could not parse int from %s", rpcResponse.Result)
	}

	return val, nil
}

func (rpcResponse *RPCResponse) GetObject(toStruct interface{}) error {
	js, err := json.Marshal(rpcResponse.Result)
	if err != nil {
		return err
	}

	err = json.Unmarshal(js, toStruct)
	if err != nil {
		return err
	}

	return nil
}

// Package jsonrpc provides a JSON-RPC 2.0 client that sends JSON-RPC requests and receives JSON-RPC responses using HTTP.
package jsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"encoding/base64"
)

const (
	jsonrpcVersion = "2.0"
	defaultID      = 1
)

// Request represents a JSON-RPC request object.
//
// See: http://www.jsonrpc.org/specification#request_object
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      uint        `json:"id,omitempty"`
}

// RPCResponse represents a JSON-RPC response object.
// If no rpc specific error occurred Error field is nil.
//
// See: http://www.jsonrpc.org/specification#response_object
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      uint        `json:"id"`
}

// RPCError represents a JSON-RPC error object if an RPC error occurred.
//
// See: http://www.jsonrpc.org/specification#error_object
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func (e *RPCError) Error() string {
	return strconv.Itoa(e.Code) + ":" + e.Message
}

// RPCClient sends JSON-RPC requests over HTTP to the provided JSON-RPC backend.
// RPCClient is created using the factory function NewClient().
type RPCClient struct {
	endpoint      string
	httpClient    *http.Client
	customHeaders map[string]string
}

// NewClient returns a new RPCClient instance with default configuration.
// Endpoint is the JSON-RPC service url to which JSON-RPC requests are sent.
func NewClient(endpoint string) *RPCClient {
	return &RPCClient{
		endpoint:      endpoint,
		httpClient:    &http.Client{},
		customHeaders: make(map[string]string),
	}
}

// SetCustomHeaders is used to set a list of custom headers for each RPC request.
// You could for example set the Authorization Bearer here.
func (client *RPCClient) SetCustomHeaders(headers map[string]string) {
	for k, v := range headers {
		client.customHeaders[k] = v
	}
}

// SetBasicAuth is a helper function that sets the header for the given basic authentication credentials.
// To reset / disable authentication just set username or password to an empty string value.
func (client *RPCClient) SetBasicAuth(username string, password string) {
	if username == "" && password == "" {
		delete(client.customHeaders, "Authorization")
		return
	}
	auth := username + ":" + password
	client.customHeaders["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// SetHTTPClient can be used to set a custom http.Client.
// This can be useful for example if you want to customize the http.Client behaviour (e.g. proxy or tls settings)
func (client *RPCClient) SetHTTPClient(httpClient *http.Client) {
	client.httpClient = httpClient
}

// Call sends a JSON-RPC request over HTTP to the JSON-RPC service url.
//
// Usage:
// Call("getinfo") no parameters
// Call("setPerson", "Alex", 1, 2) if more than one parameter is provided, they are automatically wrapped into an array
// Call("setTime", "11:30") if one parameter is provided, it is wrapped into an array if it is a primitive value type (int, string, etc.)
// Call("setNumbers", []int{1, 2, 3}) -> "params": [1, 2, 3]
// Call("setPerson", &Person{Name: "Alex", "Age": 35}) -> "params": {"name":"Alex","age":35}
// Call("setPersons", []*Person{&Person{Name: "Alex", "Age": 35}) -> "params": [{"name":"Alex","age":35}]
// Call("explicitNull", nil)  -> "params": [null]
// Call("emptyArray", []interface{}{}) -> "params": []
// Call("emptyObject", struct{}{}) -> "params": {}
//
// If something went wrong on the network / http level or if json parsing failed, error != nil is returned.
//
// If something went wrong on the rpc-service / protocol level the Error field of the returned RPCResponse is set
// and contains information about the error.
//
// If the request was successful the Error field is nil and the Result field of the Response struct contains the rpc result.
func (client *RPCClient) Call(method string, params ...interface{}) (*RPCResponse, error) {

	request := &RPCRequest{
		ID:      defaultID,
		Method:  method,
		Params:  transformParams(params...),
		JSONRPC: jsonrpcVersion,
	}

	return client.doCall(request)
}


// CallFor does the same as Call() but you can directly provide a result object.
// If something went wrong an error is returned, otherwise your out parameter holds the result.
//
// The out parameter behaves exactly as if it was used in json.Unmarshal().
//
// You won't get an RPCResponse object. But error is of type *RPCError if an rpc error occurred.
func (client *RPCClient) CallFor(out interface{}, method string, params ...interface{}) error {
	rpcResponse, err := client.Call(method, params...)
	if err != nil {
		return err
	}

	if rpcResponse.Error != nil {
		return rpcResponse.Error
	}

	err = rpcResponse.GetObject(out)
	if err != nil {
		return err
	}

	return nil
}

func transformParams(params ...interface{}) interface{} {
	var finalParams interface{}

	// if params was nil skip this and p stays nil
	if params != nil {
		switch len(params) {
		case 0: // no parameters were provided, do nothing so finalParam is nil and will be omitted
		case 1: // one param was provided, use it directly as is, or wrap primitive types in array
			if params[0] != nil {
				var typeOf reflect.Type

				// traverse until nil or not a pointer type
				for typeOf = reflect.TypeOf(params[0]); typeOf != nil && typeOf.Kind() == reflect.Ptr; typeOf = typeOf.Elem() {
				}

				if typeOf != nil {
					// now check if we can directly marshal the type or if it must be wrapped in an array
					switch typeOf.Kind() {
					// for these types we just do nothing, since value of p is already unwrapped from the array params
					case reflect.Struct:
						finalParams = params[0]
					case reflect.Array:
						finalParams = params[0]
					case reflect.Slice:
						finalParams = params[0]
					case reflect.Interface:
						finalParams = params[0]
					case reflect.Map:
						finalParams = params[0]
					default: // everything else must stay in an array (int, string, etc)
						finalParams = params
					}
				}
			} else {
				finalParams = params
			}
		default: // if more than one parameter was provided it should be treated as an array
			finalParams = params
		}
	}

	return finalParams
}

func (client *RPCClient) doCall(rpcRequest *RPCRequest) (*RPCResponse, error) {

	httpRequest, err := client.newRequest(rpcRequest)
	if err != nil {
		return nil, fmt.Errorf("%v on %v: %v", rpcRequest.Method, httpRequest.URL.String(), err.Error())
	}
	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("%v on %v: %v", rpcRequest.Method, httpRequest.URL.String(), err.Error())
	}
	defer httpResponse.Body.Close()

	var rpcResponse *RPCResponse
	decoder := json.NewDecoder(httpResponse.Body)
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	err = decoder.Decode(&rpcResponse)

	if err != nil {
		return nil, fmt.Errorf("rpc call %v() on %v status code: %v. could not decode body to rpc response: %v", rpcRequest.Method, httpRequest.URL.String(), httpResponse.StatusCode, err.Error())
	}

	if rpcResponse == nil {
		return nil, fmt.Errorf("rpc call %v() on %v status code: %v. unable to decode body to rpc response object", rpcRequest.Method, httpRequest.URL.String(), httpResponse.StatusCode)
	}

	return rpcResponse, nil
}

func (client *RPCClient) newRequest(req interface{}) (*http.Request, error) {

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", client.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	// set default headers first, so that even content type and accept can be overwritten
	for k, v := range client.customHeaders {
		request.Header.Set(k, v)
	}

	return request, nil
}

// GetInt64 converts the rpc response to an int64 and returns it.
//
// If result was not an integer an error is returned.
func (rpcResponse *RPCResponse) GetInt() (int64, error) {
	val, ok := rpcResponse.Result.(json.Number)
	if !ok {
		return 0, fmt.Errorf("could not parse int64 from %s", rpcResponse.Result)
	}

	i, err := val.Int64()
	if err != nil {
		return 0, err
	}

	return i, nil
}

// GetFloat64 converts the rpc response to an float64 and returns it.
//
// If result was not an float64 an error is returned.
func (rpcResponse *RPCResponse) GetFloat() (float64, error) {
	val, ok := rpcResponse.Result.(json.Number)
	if !ok {
		return 0, fmt.Errorf("could not parse float64 from %s", rpcResponse.Result)
	}

	f, err := val.Float64()
	if err != nil {
		return 0, err
	}

	return f, nil
}

// GetBool converts the rpc response to a bool and returns it.
//
// If result was not a bool an error is returned.
func (rpcResponse *RPCResponse) GetBool() (bool, error) {
	val, ok := rpcResponse.Result.(bool)
	if !ok {
		return false, fmt.Errorf("could not parse bool from %s", rpcResponse.Result)
	}

	return val, nil
}

// GetString converts the rpc response to a string and returns it.
//
// If result was not a string an error is returned.
func (rpcResponse *RPCResponse) GetString() (string, error) {
	val, ok := rpcResponse.Result.(string)
	if !ok {
		return "", fmt.Errorf("could not parse string from %s", rpcResponse.Result)
	}

	return val, nil
}

// GetObject converts the rpc response to an arbitrary type.
//
// The function works as you would expect it from json.Unmarshal()
func (rpcResponse *RPCResponse) GetObject(toType interface{}) error {
	js, err := json.Marshal(rpcResponse.Result)
	if err != nil {
		return err
	}

	err = json.Unmarshal(js, toType) //TODO: toType vs &toType
	if err != nil {
		return err
	}

	return nil
}

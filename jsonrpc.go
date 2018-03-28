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
	"io/ioutil"
)

const (
	jsonrpcVersion = "2.0"
	defaultID      = 1
)

// HTTPError is returned if an error on HTTP layer occurred.
// This is helpful for further error investigation (e.g. check for status code 403)
type HTTPError struct {
	Status int
	URL    string
	Body   []byte
	error  string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("error: status %v on POST %v", e.Status, e.URL)
}

// Request represents a JSON-RPC request object.
//
// See: http://www.jsonrpc.org/specification#request_object
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      uint        `json:"id,omitempty"`
}

// Response represents a JSON-RPC response object.
// If no rpc specific error occurred Error field is nil.
//
// See: http://www.jsonrpc.org/specification#response_object
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      uint        `json:"id"`
}

// Error represents a JSON-RPC error object if an RPC error occurred.
//
// See: http://www.jsonrpc.org/specification#error_object
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func (e *RPCError) Error() string {
	return strconv.Itoa(e.Code) + ": " + e.Message
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
func (client *RPCClient) SetHTTPClient(httpClient *http.Client) { // TODO: pointer vs struct
	client.httpClient = httpClient
}

// Call sends a JSON-RPC request over HTTP to the JSON-RPC service url.
//
// Usage:
// Call("getinfo") no parameters
// Call("setPerson", "Alex", 1, 2) if more than one parameter is provided, they are automatically wrapped into an array
// Call("setTime", "11:30") if one parameter is provided, it is wrapped into an array if it is a primitive value type (int, string, etc)
// Call("setNumbers", []int{1, 2, 3}) positional parameters can be set directly
// Call("setNumbers", &Person{Name: "Alex", "Age": 35})
// Call("setNumbers", []*Person{&Person{Name: "Alex", "Age": 35}) object wrapped in array
// Call("explicitNull", nil)
// Call("emptyArray", []interface{}{}) empty array
// Call("emptyObject", {}interface{}{ empty array // TODO:
//
// If something went wrong on the network / http level or if json parsing failed, error != nil is returned.
// If an HTTP error occurred the error is of type HTTPError, to investigate the error code.
//
// If something went wrong on the rpc-service / protocol level the Error field of the returned Response is set
// and contains information about the error.
//
// If the request was successful the Error field is nil and the Result field of the Response struct contains the rpc result.
func (client *RPCClient) Call(method string, params ...interface{}) (*RPCResponse, error) {
	var finalParam interface{}

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
						finalParam = params[0]
					case reflect.Array:
						finalParam = params[0]
					case reflect.Slice:
						finalParam = params[0]
					case reflect.Interface:
						finalParam = params[0]
					case reflect.Map:
						finalParam = params[0]
					default: // everything else must stay in an array (int, string, etc)
						finalParam = params
					}
				}
			} else {
				finalParam = params
			}
		default: // if more than one parameter was provided it should be treated as an array
			finalParam = params
		}
	}

	request := &RPCRequest{
		ID:      defaultID,
		Method:  method,
		Params:  finalParam,
		JSONRPC: jsonrpcVersion,
	}

	httpRequest, err := client.newRequest(request)
	if err != nil {
		return nil, err
	}
	return client.doCall(httpRequest)
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

type BatchRequest struct {
}

type RPCCall struct {
	Method string
	Params interface{}
}

func GetRequest(method string, params ...interface{}) *RPCCall {

}

func (client *RPCClient) CallBatch(batchObjects []RPCCall) ([]*RPCResponse, error) {
	requests := make([]*RPCRequest, len(batchObjects))
	for i, obj := range batchObjects {
		// if params was nil skip this and p stays nil
		var finalParam interface{}
		if obj.Params != nil {
			finalParam = obj.Params
			var typeOf reflect.Type

			// traverse until nil or not a pointer type
			for typeOf = reflect.TypeOf(finalParam); typeOf != nil && typeOf.Kind() == reflect.Ptr; typeOf = typeOf.Elem() {
			}

			if typeOf != nil {
				// now check if we can directly marshal the type or if it must be wrapped in an array
				switch typeOf.Kind() {
				// for these types we just do nothing, since value of p is already unwrapped from the array params
				case reflect.Struct:
				case reflect.Array:
				case reflect.Slice:
				case reflect.Interface:
				case reflect.Map:
				default: // everything else must stay in an array
					finalParam = []interface{}{obj.Params}
				}
			}
		}

		requests[i] = &RPCRequest{
			ID:      uint(i),
			Method:  obj.Method,
			Params:  finalParam,
			JSONRPC: jsonrpcVersion,
		}
	}

	httpRequest, err := client.newRequest(requests)
	if err != nil {
		return err
	}
	return client.doCallBatch(httpRequest)
}

func (client *RPCClient) doCall(req *http.Request) (*RPCResponse, error) {
	httpResponse, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode >= 400 {
		data, err := ioutil.ReadAll(httpResponse.Body)
		if err != nil {
			return nil, &HTTPError{
				Status: httpResponse.StatusCode,
				URL:    req.URL.String(),
				Body:   nil,
			}
		}
		return nil, &HTTPError{
			Status: httpResponse.StatusCode,
			URL:    req.URL.String(),
			Body:   data,
		}

	}

	var rpcResponse *RPCResponse
	decoder := json.NewDecoder(httpResponse.Body)
	decoder.UseNumber()
	err = decoder.Decode(&rpcResponse)
	if err != nil {
		return nil, err
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

func (client *RPCClient) newBatchRequest(requests ...interface{}) (*http.Request, error) {

	body, err := json.Marshal(requests)
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

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")

	return request, nil
}

// GetInt converts the rpc response to an int and returns it.
//
// This is a convenient function. Int could be 32 or 64 bit, depending on the architecture the code is running on.
// For a deterministic result use GetInt64().
//
// If result was not an integer an error is returned.
func (rpcResponse *RPCResponse) GetInt() (int, error) {
	i, err := rpcResponse.GetInt64()
	return int(i), err
}

// GetInt64 converts the rpc response to an int64 and returns it.
//
// If result was not an integer an error is returned.
func (rpcResponse *RPCResponse) GetInt64() (int64, error) {
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
func (rpcResponse *RPCResponse) GetFloat64() (float64, error) {
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

	err = json.Unmarshal(js, toType)
	if err != nil {
		return err
	}

	return nil
}

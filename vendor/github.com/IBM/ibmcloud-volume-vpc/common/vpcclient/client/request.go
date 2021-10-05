/**
 * Copyright 2020 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package client ...
package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client/payload"
	"github.com/fatih/structs"
)

// Operation defines the API operation to be invoked
type Operation struct {
	Name        string
	Method      string
	PathPattern string
}

// Request defines the properties of an API request. It can then be invoked to
// call the underlying API specified by the supplied operation
type Request struct {
	httpClient    *http.Client
	baseURL       string
	authenHandler handler

	context context.Context

	operation  *Operation
	pathParams Params
	headers    http.Header

	debugWriter io.Writer

	queryValues     url.Values
	bodyProvider    BodyProvider
	successConsumer ResponseConsumer
	errorConsumer   ResponseConsumer
	resourceGroup   string
}

// BodyProvider declares an interface that describes an HTTP body, for
// both request and response
type BodyProvider interface {
	ContentType() string
	Body() (io.Reader, error)
}

// ResponseConsumer ...
type ResponseConsumer interface {
	Consume(io.Reader) error
	Receiver() interface{}
}

func (r *Request) path() string {
	path := r.operation.PathPattern
	for k, v := range r.pathParams {
		path = strings.Replace(path, "{"+k+"}", v, -1)
	}
	return path
}

// URL constructs the full URL for a request
func (r *Request) URL() string {
	baseURL, baseErr := url.Parse(r.baseURL)
	if baseErr != nil {
		return ""
	}
	if !strings.HasSuffix(baseURL.Path, "/") {
		baseURL.Path += "/"
	}
	pathURL, pathErr := url.Parse(r.path())
	if pathErr != nil {
		return ""
	}
	resolvedURL := baseURL.ResolveReference(pathURL)
	resolvedURL.RawQuery = r.queryValues.Encode()

	return resolvedURL.String()
}

// PathParameter sets a path parameter to be resolved on invocation of a request
func (r *Request) PathParameter(name, value string) *Request {
	r.pathParams[name] = value
	return r
}

// AddQueryValue ...
func (r *Request) AddQueryValue(key, value string) *Request {
	if r.queryValues == nil {
		r.queryValues = url.Values{}
	}
	r.queryValues.Add(key, value)
	return r
}

// SetQueryValue ...
func (r *Request) SetQueryValue(key, value string) *Request {
	if r.queryValues == nil {
		r.queryValues = url.Values{}
	}
	r.queryValues.Set(key, value)
	return r
}

// JSONBody converts the supplied argument to JSON to use as the body of a request
func (r *Request) JSONBody(p interface{}) *Request {
	if r.operation.Method == http.MethodPost && reflect.ValueOf(p).Kind() == reflect.Struct {
		structs.DefaultTagName = "json"
		m := structs.Map(p)

		if r.resourceGroup != "" {
			m["resourceGroup"] = r.resourceGroup
		}

		r.bodyProvider = payload.NewJSONBodyProvider(m)
	} else {
		r.bodyProvider = payload.NewJSONBodyProvider(p)
	}
	return r
}

// MultipartFileBody configures the POST payload to be sent in multi-part format. The
// content is read from the supplied Reader.
func (r *Request) MultipartFileBody(name string, contents io.Reader) *Request {
	r.bodyProvider = payload.NewMultipartFileBody(name, contents)
	return r
}

// JSONSuccess configures the receiver to use to process a JSON response
// for a successful (2xx) response
func (r *Request) JSONSuccess(receiver interface{}) *Request {
	r.successConsumer = payload.NewJSONConsumer(receiver)
	return r
}

// JSONError configures the error to populate in the event of an unsuccessful
// (non-2xx) response
func (r *Request) JSONError(receiver error) *Request {
	r.errorConsumer = payload.NewJSONConsumer(receiver)
	return r
}

// Invoke performs the request, and populates the response or error as appropriate
func (r *Request) Invoke() (*http.Response, error) {
	err := r.authenHandler.Before(r)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if r.bodyProvider != nil {
		body, err = r.bodyProvider.Body()
		if err != nil {
			return nil, err
		}

		if contentType := r.bodyProvider.ContentType(); contentType != "" {
			r.headers.Set("Content-Type", contentType)
		}
	}

	httpRequest, err := http.NewRequest(r.operation.Method, r.URL(), body)
	if err != nil {
		return nil, err
	}

	for k, v := range r.headers {
		httpRequest.Header[k] = v
	}

	r.debugRequest(httpRequest)

	resp, err := r.httpClient.Do(httpRequest.WithContext(r.context))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	r.debugResponse(resp)

	switch {
	case resp.StatusCode == http.StatusNoContent:
		break

	case resp.StatusCode >= 200 && resp.StatusCode <= 299:
		if r.successConsumer != nil {
			err = r.successConsumer.Consume(resp.Body)
		}

	default:
		if r.errorConsumer != nil {
			err = r.errorConsumer.Consume(resp.Body)
			if err == nil {
				err = r.errorConsumer.Receiver().(error)
			}
		}
	}

	return resp, err
}

func (r *Request) debugRequest(req *http.Request) {
	if r.debugWriter == nil {
		return
	}

	multipart := strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data")
	dumpedRequest, err := httputil.DumpRequest(req, !multipart)
	if err != nil {
		r.debugf("Error dumping request\n%s\n", err)
		return
	}

	r.debugf("\nREQUEST: [%s]\n%s\n", time.Now().Format(time.RFC3339), sanitize(dumpedRequest))
	if multipart {
		r.debugf("[MULTIPART/FORM-DATA CONTENT HIDDEN]\n")
	}
}

func (r *Request) debugResponse(resp *http.Response) {
	if r.debugWriter == nil {
		return
	}

	dumpedResponse, err := httputil.DumpResponse(resp, true)
	if err != nil {
		fmt.Fprintf(r.debugWriter, "Error dumping response\n%s\n", err)
		return
	}

	r.debugf("\nRESPONSE: [%s]\n%s\n", time.Now().Format(time.RFC3339), sanitize(dumpedResponse))
}

func (r *Request) debugf(format string, args ...interface{}) {
	fmt.Fprintf(r.debugWriter, format, args...)
}

// RedactedFillin used as a replacement string in debug logs for sensitive data
const RedactedFillin = "[REDACTED]"

func sanitize(input []byte) string {
	sanitized := string(input)

	re := regexp.MustCompile(`(?mi)^Authorization: .*`)
	sanitized = re.ReplaceAllString(sanitized, "Authorization: "+RedactedFillin)

	re = regexp.MustCompile(`(?mi)^X-Auth-Token: .*`)
	sanitized = re.ReplaceAllString(sanitized, "X-Auth-Token: "+RedactedFillin)

	re = regexp.MustCompile(`(?mi)^APIKey: .*`)
	sanitized = re.ReplaceAllString(sanitized, "APIKey: "+RedactedFillin)

	sanitized = sanitizeJSON("key", sanitized)
	sanitized = sanitizeJSON("password", sanitized)
	sanitized = sanitizeJSON("passphrase", sanitized)

	return sanitized
}

func sanitizeJSON(propertySubstring string, json string) string {
	regex := regexp.MustCompile(fmt.Sprintf(`(?i)"([^"]*%s[^"]*)":\s*"[^\,]*"`, propertySubstring))
	return regex.ReplaceAllString(json, fmt.Sprintf(`"$1":"%s"`, RedactedFillin))
}

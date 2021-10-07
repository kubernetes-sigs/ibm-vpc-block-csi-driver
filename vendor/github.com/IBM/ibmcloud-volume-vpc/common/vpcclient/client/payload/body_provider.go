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

// Package payload ...
package payload

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
)

// JSONBodyProvider ...
type JSONBodyProvider struct{ payload interface{} }

// NewJSONBodyProvider ...
func NewJSONBodyProvider(p interface{}) *JSONBodyProvider {
	return &JSONBodyProvider{payload: p}
}

// ContentType ...
func (p *JSONBodyProvider) ContentType() string {
	return "application/json"
}

// Body ...
func (p *JSONBodyProvider) Body() (io.Reader, error) {
	buf := &bytes.Buffer{}

	err := json.NewEncoder(buf).Encode(p.payload)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// MultipartFileBody ...
type MultipartFileBody struct {
	name            string
	contents        io.Reader
	multipartWriter *multipart.Writer
	pipeReader      *io.PipeReader
	pipeWriter      *io.PipeWriter
}

// NewMultipartFileBody ...
func NewMultipartFileBody(name string, contents io.Reader) *MultipartFileBody {
	pr, pw := io.Pipe()
	return &MultipartFileBody{
		name:            name,
		contents:        contents,
		pipeReader:      pr,
		pipeWriter:      pw,
		multipartWriter: multipart.NewWriter(pw),
	}
}

// ContentType ...
func (p *MultipartFileBody) ContentType() string {
	return p.multipartWriter.FormDataContentType()
}

// Body ...
func (p *MultipartFileBody) Body() (io.Reader, error) {
	go p.copyBody()
	return p.pipeReader, nil
}

func (p *MultipartFileBody) copyBody() {
	defer p.Close()

	fileWriter, err := p.multipartWriter.CreateFormFile(p.name, "image")
	if err != nil {
		p.pipeWriter.CloseWithError(err)
	}

	_, err = io.Copy(fileWriter, p.contents)
	if err != nil {
		p.pipeWriter.CloseWithError(err)
	}
}

// Close ...
func (p *MultipartFileBody) Close() {
	_ = p.multipartWriter.Close()
	_ = p.pipeWriter.Close()
}

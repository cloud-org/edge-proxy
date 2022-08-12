/*
Copyright 2020 The OpenYurt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"

	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

// ReqInfoString formats a string for request info
func ReqInfoString(info *apirequest.RequestInfo) string {
	if info == nil {
		return ""
	}

	return fmt.Sprintf("%s %s for %s", info.Verb, info.Resource, info.Path)
}

// NewDualReadCloser create an dualReadCloser object
func NewDualReadCloser(req *http.Request, rc io.ReadCloser, isRespBody bool) (io.ReadCloser, io.ReadCloser) {
	pr, pw := io.Pipe()
	dr := &dualReadCloser{
		req:        req,
		rc:         rc,
		pw:         pw,
		isRespBody: isRespBody,
	}

	return dr, pr
}

type dualReadCloser struct {
	req *http.Request
	rc  io.ReadCloser
	pw  *io.PipeWriter
	// isRespBody shows rc(is.ReadCloser) is a response.Body
	// or not(maybe a request.Body). if it is true(it's a response.Body),
	// we should close the response body in Close func, else not,
	// it(request body) will be closed by http request caller
	isRespBody bool
}

// Read read data into p and write into pipe
func (dr *dualReadCloser) Read(p []byte) (n int, err error) {

	n, err = dr.rc.Read(p)
	if n > 0 {
		if n, err := dr.pw.Write(p[:n]); err != nil {
			klog.Errorf("dualReader: failed to write %v", err)
			return n, err
		}
	}

	return
}

// Close close two readers
func (dr *dualReadCloser) Close() error {
	errs := make([]error, 0)
	if dr.isRespBody {
		if err := dr.rc.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if err := dr.pw.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		return fmt.Errorf("failed to close dualReader, %v", errs)
	}

	return nil
}

// gzipReaderCloser will gunzip the data if response header
// contains Content-Encoding=gzip header.
type gzipReaderCloser struct {
	body io.ReadCloser
	zr   *gzip.Reader
	zerr error
}

func (grc *gzipReaderCloser) Read(b []byte) (n int, err error) {
	if grc.zerr != nil {
		return 0, grc.zerr
	}

	if grc.zr == nil {
		grc.zr, err = gzip.NewReader(grc.body)
		if err != nil {
			grc.zerr = err
			return 0, err
		}
	}

	return grc.zr.Read(b)
}

func (grc *gzipReaderCloser) Close() error {
	return grc.body.Close()
}

func NewGZipReaderCloser(header http.Header, body io.ReadCloser, info *apirequest.RequestInfo, caller string) (io.ReadCloser, bool) {
	if header.Get("Content-Encoding") != "gzip" {
		return body, false
	}

	klog.Infof("response of %s will be ungzip at %s", ReqInfoString(info), caller)
	return &gzipReaderCloser{
		body: body,
	}, true
}

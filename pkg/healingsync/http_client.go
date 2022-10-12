/*
@Copyright (C) Ctyun Inc. All rights reserved.
@Date : 2022/9/29 14:59
@Author : linshw
@Descriptions ：
*/

package healingsync

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// 需要调用方response.Body.Close()
func httpRequest(url string) (*http.Response, error) {
	var netTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	var client = &http.Client{
		Timeout:   time.Second * 30,
		Transport: netTransport,
	}

	return client.Get(url)
}

// GetBodyData 处理gzip压缩
func getBodyData(response *http.Response) (body []byte, err error) {
	var reader io.ReadCloser
	defer func() {
		_ = reader.Close()
	}()

	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(response.Body)
		if err != nil {
			return nil, err
		}
	default:
		reader = response.Body
	}
	body, err = ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	response.Body = ioutil.NopCloser(bytes.NewReader(body))
	return
}

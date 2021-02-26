package ipp

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
)

type HttpAdapter struct {
	host     string
	port     int
	username string
	password string
	useTLS   bool
	client   *http.Client
}

func NewHttpAdapter(host string, port int, username, password string, useTLS bool) *HttpAdapter {
	httpClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return &HttpAdapter{
		host:     host,
		port:     port,
		username: username,
		password: password,
		useTLS:   useTLS,
		client:   &httpClient,
	}
}

func (h *HttpAdapter) SendRequest(url string, req *Request, additionalResponseData io.Writer) (*Response, error) {
	payload, err := req.Encode()
	if err != nil {
		return nil, err
	}

	var body io.Reader
	size := len(payload)

	if req.File != nil && req.FileSize != -1 {
		size += req.FileSize

		body = io.MultiReader(bytes.NewBuffer(payload), req.File)
	} else {
		body = bytes.NewBuffer(payload)
	}

	httpReq, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Length", strconv.Itoa(size))
	httpReq.Header.Set("Content-Type", ContentTypeIPP)

	if h.username != "" && h.password != "" {
		httpReq.SetBasicAuth(h.username, h.password)
	}

	httpResp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
		return nil, HTTPError{
			Code: httpResp.StatusCode,
		}
	}

	resp, err := NewResponseDecoder(httpResp.Body).Decode(additionalResponseData)
	if err != nil {
		return nil, err
	}

	err = resp.CheckForErrors()
	return resp, err
}

func (h *HttpAdapter) GetHttpUri(namespace string, object interface{}) string {
	proto := "http"
	if h.useTLS {
		proto = "https"
	}

	uri := fmt.Sprintf("%s://%s:%d", proto, h.host, h.port)

	if namespace != "" {
		uri = fmt.Sprintf("%s/%s", uri, namespace)
	}

	if object != nil {
		uri = fmt.Sprintf("%s/%v", uri, object)
	}

	return uri
}

func (h *HttpAdapter) TestConnection() error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", h.host, h.port))
	if err != nil {
		return err
	}
	conn.Close()

	return nil
}

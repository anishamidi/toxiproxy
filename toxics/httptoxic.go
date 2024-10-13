package toxics

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
)

// HttpToxic modifies HTTP responses with configurable status code, headers, and body.
type HttpToxic struct {
	StatusCode      int               `json:"status_code"`
	ResponseHeaders map[string]string `json:"response_headers"`
	ResponseBody    string            `json:"response_body"`
}

func (t *HttpToxic) GetBufferSize() int {
	return 1024
}

// ModifyResponse updates the HTTP response with the configured status code, headers, and body.
func (t *HttpToxic) ModifyResponse(resp *http.Response) {
	for key, value := range t.ResponseHeaders {
		resp.Header.Set(key, value)
	}
	resp.StatusCode = t.StatusCode
	resp.Body = io.NopCloser(bytes.NewBufferString(t.ResponseBody))
	resp.ContentLength = int64(len(t.ResponseBody))
}

// Pipe intercepts the HTTP response stream, modifies it, and then sends it through.
func (t *HttpToxic) Pipe(stub *ToxicStub) {
	for {
		select {
		case <-stub.Interrupt:
			return
		case c := <-stub.Input:
			if c == nil {
				stub.Close()
				return
			}

			// Read the HTTP response and modify it
			buffer := bytes.NewBuffer(make([]byte, 0, 32*1024))
			tee := io.TeeReader(bytes.NewReader(c.Data), buffer)
			resp, err := http.ReadResponse(bufio.NewReader(tee), nil)
			if err == nil {
				t.ModifyResponse(resp)
				var modifiedBuffer bytes.Buffer
				resp.Write(&modifiedBuffer)
				c.Data = modifiedBuffer.Bytes()
			}

			// Send the modified data downstream
			stub.Output <- c
		}
	}
}

// parseHeaders parses headers from a semicolon-separated string.
func parseHeaders(headerString string) map[string]string {
	headers := map[string]string{}
	if len(headerString) == 0 {
		return headers
	}
	pairs := strings.Split(headerString, ";")
	for _, pair := range pairs {
		splitPair := strings.SplitN(pair, ":", 2)
		if len(splitPair) == 2 {
			key := strings.TrimSpace(splitPair[0])
			value := strings.TrimSpace(splitPair[1])
			headers[key] = value
		}
	}
	return headers
}

// init registers the HttpToxic with Toxiproxy.

func init() {
	Register("httptoxic", new(HttpToxic))
}

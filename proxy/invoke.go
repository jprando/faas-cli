// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package proxy

import (
	"bytes"

	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// InvokeFunction a function
func InvokeFunction(gateway string, name string, bytesIn *[]byte, contentType string, query []string, headers []string) (*[]byte, error) {
	var resBytes []byte

	gateway = strings.TrimRight(gateway, "/")

	reader := bytes.NewReader(*bytesIn)

	var timeout *time.Duration
	client := MakeHTTPClient(timeout)

	qs, qsErr := buildQueryString(query)
	if qsErr != nil {
		return nil, qsErr
	}

	headerMap, headerErr := parseHeaders(headers)
	if headerErr != nil {
		return nil, headerErr
	}

	gatewayURL := gateway + "/function/" + name + qs

	req, err := http.NewRequest(http.MethodPost, gatewayURL, reader)
	if err != nil {
		fmt.Println()
		fmt.Println(err)
		return nil, fmt.Errorf("cannot connect to OpenFaaS on URL: %s", gateway)
	}

	req.Header.Add("Content-Type", contentType)
	// Add additional headers to request
	for name, value := range headerMap {
		req.Header.Add(name, value)
	}

	SetAuth(req, gateway)

	res, err := client.Do(req)

	if err != nil {
		fmt.Println()
		fmt.Println(err)
		return nil, fmt.Errorf("cannot connect to OpenFaaS on URL: %s", gateway)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	switch res.StatusCode {
	case http.StatusOK:
		var readErr error
		resBytes, readErr = ioutil.ReadAll(res.Body)
		if readErr != nil {
			return nil, fmt.Errorf("cannot read result from OpenFaaS on URL: %s %s", gateway, readErr)
		}
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("unauthorized access, run \"faas-cli login\" to setup authentication for this server")
	default:
		bytesOut, err := ioutil.ReadAll(res.Body)
		if err == nil {
			return nil, fmt.Errorf("server returned unexpected status code: %d - %s", res.StatusCode, string(bytesOut))
		}
	}

	return &resBytes, nil
}

func buildQueryString(query []string) (string, error) {
	qs := ""

	if len(query) > 0 {
		qs = "?"
		for _, queryValue := range query {
			qs = qs + queryValue + "&"
			if strings.Contains(queryValue, "=") == false {
				return "", fmt.Errorf("the --query flags must take the form of key=value (= not found)")
			}
			if strings.HasSuffix(queryValue, "=") {
				return "", fmt.Errorf("the --query flag must take the form of: key=value (empty value given, or value ends in =)")
			}
		}
		qs = strings.TrimRight(qs, "&")
	}

	return qs, nil
}

// parseHeaders parses header values from command
func parseHeaders(headers []string) (map[string]string, error) {
	headerMap := make(map[string]string)

	for _, header := range headers {
		headerValues := strings.Split(header, "=")
		if len(headerValues) != 2 {
			return headerMap, fmt.Errorf("the --header or -H flag must take the form of key=value")
		}

		name, value := headerValues[0], headerValues[1]
		if name == "" {
			return headerMap, fmt.Errorf("the --header or -H flag must take the form of key=value (empty key given)")
		}

		if value == "" {
			return headerMap, fmt.Errorf("the --header or -H flag must take the form of key=value (empty value given)")
		}

		headerMap[name] = value
	}
	return headerMap, nil
}

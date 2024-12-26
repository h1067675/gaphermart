package client

import (
	"io"
	"net/http"
	"strconv"
)

type Client struct {
}

func (c *Client) GET(server string, endpoint string, order int) ([]byte, int, error) {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodGet, server+endpoint+strconv.Itoa(order), nil)
	if err != nil {
		return nil, 0, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, 0, err
	}
	return body, response.StatusCode, nil
}

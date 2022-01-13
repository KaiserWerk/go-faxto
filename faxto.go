package faxto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	rawBaseUrl = "https://fax.to/api/v2$action$?api_key=$key$"
)

type (
	balanceResponse struct {
		Status                      string  `json:"status"`
		Balance                     float64 `json:"balance"`
		SubscriptionCreditAllowance float64 `json:"sunbscription_credit_allowance"`
	}
	faxCostResponse struct {
		Status string  `json:"status"`
		Cost   float64 `json:"cost"`
	}
	faxStatusResponse struct {
		Status  string  `json:"status"`
		Message string  `json:"message"`
		Balance float64 `json:"user_cash_balance"`
	}
	FaxHistoryEntry struct {
		Id      uint64 `json:"id"`
		Created struct {
			Date         time.Time `json:"date"`
			DateTimeZone uint8     `json:"datetime_zone"`
			Timezone     string    `json:"timezone"`
		} `json:"created"`
		DocumentId uint64 `json:"document_id"`
		Document   string `json:"document"`
		Recipient  string `json:"recipient"`
		Status     string `json:"status"`
	}
	faxHistoryResponse struct {
		Status  string            `json:"status"`
		History []FaxHistoryEntry `json:"history"`
	}
	fileUploadResponse struct {
		Status     string `json:"status"`
		DocumentId uint64 `json:"document_id"`
		TotalPages uint64 `json:"total_pages"`
	}

	Client struct {
		baseUrl    string
		httpClient *http.Client
	}
	File struct {
		Id       uint64    `json:"id"`
		Filename string    `json:"filename"`
		Pages    uint      `json:"pages"`
		Size     uint64    `json:"size"`
		Uploaded time.Time `json:"uploaded"`
	}
)

func NewClient(apiKey string) Client {
	return Client{
		baseUrl:    strings.ReplaceAll(rawBaseUrl, "$key$", apiKey),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) SendFax(number string, fileId uint64) error {
	data := url.Values{
		"fax_number":  {number},
		"document_id": {fmt.Sprintf("%d", fileId)},
	}

	resp, err := c.httpClient.PostForm(strings.ReplaceAll(c.baseUrl, "$action$", "/fax"), data)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("expected status < 400, got %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) GetBalance() (float64, error) {
	req, err := http.NewRequest(http.MethodGet, strings.ReplaceAll(c.baseUrl, "$action$", "/balance"), nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("expected status < 400, got %d", resp.StatusCode)
	}

	var b balanceResponse
	err = json.NewDecoder(resp.Body).Decode(&b)
	if err != nil {
		return 0, err
	}

	if b.Status != "success" {
		return 0, fmt.Errorf("expected status 'success', got '%s'", b.Status)
	}

	return b.Balance, nil
}

func (c *Client) GetFaxCost(number string, docId uint64) (float64, error) {
	req, err := http.NewRequest(http.MethodGet, strings.ReplaceAll(c.baseUrl, "$action$", fmt.Sprintf("/fax/%d/costs", docId))+"&fax_number="+number, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("expected status < 400, got %d", resp.StatusCode)
	}

	var cost faxCostResponse
	err = json.NewDecoder(resp.Body).Decode(&cost)
	if err != nil {
		return 0, err
	}

	return cost.Cost, nil
}

func (c *Client) GetFaxStatus(faxJobId int) (string, error) {
	req, err := http.NewRequest(http.MethodGet, strings.ReplaceAll(c.baseUrl, "$action$", fmt.Sprintf("/fax/%d/status", faxJobId)), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("expected status < 400, got %d", resp.StatusCode)
	}

	var fs faxStatusResponse
	err = json.NewDecoder(resp.Body).Decode(&fs)
	if err != nil {
		return "", err
	}

	return fs.Status, nil
}

func (c *Client) GetFaxHistory() ([]FaxHistoryEntry, error) {
	req, err := http.NewRequest(http.MethodGet, strings.ReplaceAll(c.baseUrl, "$action$", "/fax-history"), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("expected status < 400, got %d", resp.StatusCode)
	}

	var fh faxHistoryResponse
	err = json.NewDecoder(resp.Body).Decode(&fh)
	if err != nil {
		return nil, err
	}

	if fh.Status != "success" {
		return nil, fmt.Errorf("expected status 'success', got '%s'", fh.Status)
	}

	return fh.History, nil
}

func (c *Client) UploadFile(file string) (uint64, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest(http.MethodPost, strings.ReplaceAll(c.baseUrl, "$action$", "/files"), bytes.NewBuffer(content)) // MethodPut?
	if err != nil {
		return 0, err
	}

	//req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("expected status < 400, got %d", resp.StatusCode)
	}

	var fu fileUploadResponse
	err = json.NewDecoder(resp.Body).Decode(&fu)
	if err != nil {
		return 0, err
	}

	if fu.Status != "success" {
		return 0, fmt.Errorf("expected status 'success', got '%s'", fu.Status)
	}

	return fu.DocumentId, nil
}

func (c *Client) GetFiles() ([]File, error) {
	req, err := http.NewRequest(http.MethodGet, strings.ReplaceAll(c.baseUrl, "$action$", "/files"), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("expected status < 400, got %d", resp.StatusCode)
	}

	files := make([]File, 0)
	err = json.NewDecoder(resp.Body).Decode(&files)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (c *Client) DeleteFile(fileId uint64) error {
	req, err := http.NewRequest(http.MethodDelete, strings.ReplaceAll(c.baseUrl, "$action$", fmt.Sprintf("/files/%d", fileId)), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("expected status < 400, got %d", resp.StatusCode)
	}

	return nil
}

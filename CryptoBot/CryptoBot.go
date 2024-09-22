package CryptoBot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
    "time"
	"strconv"
	// "bytes"
	// "log"
)

type ResultMe struct {
	AppID                        int    `json:"app_id"`
	Name                         string `json:"name"`
	PaymentProcessingBotUsername string `json:"payment_processing_bot_username"`
}

type ResponseMe struct {
	Ok     bool     `json:"ok"`
	Result ResultMe `json:"result"`
}

type ResultInvoice struct {
	InvoiceID         int     `json:"invoice_id"`
	Hash              string  `json:"hash"`
	CurrencyType      string  `json:"currency_type"`
	Asset             string  `json:"asset"`
	Amount            string  `json:"amount"`
	PayURL            string  `json:"pay_url"`
	BotInvoiceURL     string  `json:"bot_invoice_url"`
	MiniAppInvoiceURL string  `json:"mini_app_invoice_url"`
	WebAppInvoiceURL  string  `json:"web_app_invoice_url"`
	Description       string  `json:"description"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	AllowComments     bool    `json:"allow_comments"`
	AllowAnonymous    bool    `json:"allow_anonymous"`
}

type ResponseInvoice struct {
	Ok     bool          `json:"ok"`
	Result ResultInvoice `json:"result"`
}

type ResultCheck struct {
	CheckID     int       `json:"check_id"`
	Hash        string    `json:"hash"`
	Asset       string    `json:"asset"`
	Amount      string    `json:"amount"`
	BotCheckURL string    `json:"bot_check_url"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type ResponseCheck struct {
	Ok     bool   `json:"ok"`
	Result *ResultCheck `json:"result"`
	Error  *Error        `json:"error"`
}

type Error struct {
	Code int           `json:"code"`
	Name string        `json:"name"`
}

type ResponseDelete struct {
	Ok     bool   `json:"ok"`
	Result bool `json:"result"`
}

type ResponseCheckInvoice struct {
	Ok bool          `json:"ok"`
	ResultCheckInvoice ResultCheckInvoice `json:"result"`
}

type ResultCheckInvoice struct {
	Items []Item `json:"items"`
}

type Item struct {
	InvoiceID int    `json:"invoice_id"`
	Status    string `json:"status"`
}

type ResponseBalance struct {
	Ok     bool            `json:"ok"`
	Result []ResultBalance `json:"result"`
}

type ResultBalance struct {
	CurrencyCode string `json:"currency_code"`
	Available    string `json:"available"`
	OnHold       string `json:"onhold"`
}

type CryptoBotApi struct {
	apiToken string
	proxyURL string
}

func isPositiveFloat(s string) bool {
    num, err := strconv.ParseFloat(s, 64)
    if err != nil || num <= 0 {
        return false
    }
    return true
}

func hasMoreThanSixDecimalPlaces(s string) bool {
    parts := strings.Split(s, ".")
    if len(parts) == 2 {
		if len(parts[1]) > 6 {
        	return true
		}
    }
    return false
}

func CheckNumber(data string) (bool, string){
	if isPositiveFloat(data) {
		if hasMoreThanSixDecimalPlaces(data) {
			return false, "Неккоректный ввод, слишком много цифр после запятой, попробуйте снова: " + data
		} else {
			return true, "Вы ввели: " + data
		}
	}
	return false, "Неккоректный ввод, попробуйте снова: " + data
}

func NewCryptoBotApi(apiToken, proxyURL string) *CryptoBotApi {
	return &CryptoBotApi{
		proxyURL: proxyURL,
		apiToken: apiToken,
	}
}

func (b *CryptoBotApi) GetBalance() (string, bool) {
	proxyURL, err := url.Parse(b.proxyURL)
	if err != nil {
		fmt.Println("Ошибка разбора URL прокси:", err)
		return "", false
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	apiURL := "https://testnet-pay.crypt.bot/api/getBalance"

	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		fmt.Println("Ошибка создания запроса:", err)
		return "", false
	}

	req.Header.Set("Crypto-Pay-API-Token", b.apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Ошибка выполнения запроса:", err)
		return "", false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка чтения ответа:", err)
		return "", false
	}

	var response ResponseBalance
	err2 := json.Unmarshal(body, &response)
	if err2 != nil {
		fmt.Println("Error decoding JSON:", err2)
		return "", false
	}

	return response.Result[0].Available, true
}

func (b *CryptoBotApi) CheckInvoice(id int) bool {
	proxyURL, err := url.Parse(b.proxyURL)
	if err != nil {
		fmt.Println("Ошибка разбора URL прокси:", err)
		return false
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	status := "paid"
	apiURL := "https://testnet-pay.crypt.bot/api/getInvoices"
	data := url.Values{}
	data.Set("invoice_ids", strconv.Itoa(id))
	data.Set("status", status)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("Ошибка создания запроса:", err)
		return false
	}

	req.Header.Set("Crypto-Pay-API-Token", b.apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Ошибка выполнения запроса:", err)
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка чтения ответа:", err)
		return false
	}

	var response ResponseCheckInvoice
	err2 := json.Unmarshal(body, &response)
	if err2 != nil {
		fmt.Println("Error decoding JSON:", err2)
		return false
	}
	if len(response.ResultCheckInvoice.Items) > 0 {
		for i := 0; i < len(response.ResultCheckInvoice.Items); i++ {
			if response.ResultCheckInvoice.Items[i].Status == "paid" && response.ResultCheckInvoice.Items[i].InvoiceID == id {
				return true
			}
		}
	}
	return false
}

func (b *CryptoBotApi) DeleteCheck(id int) bool {
	proxyURL, err := url.Parse(b.proxyURL)
	if err != nil {
		fmt.Println("Ошибка разбора URL прокси:", err)
		return false
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	apiURL := "https://testnet-pay.crypt.bot/api/deleteCheck"
	data := url.Values{}
	data.Set("invoice_id", strconv.Itoa(id))

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("Ошибка создания запроса:", err)
		return false
	}

	req.Header.Set("Crypto-Pay-API-Token", b.apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Ошибка выполнения запроса:", err)
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка чтения ответа:", err)
		return false
	}

	var response ResponseDelete
	err2 := json.Unmarshal(body, &response)
	if err2 != nil {
		fmt.Println("Error decoding JSON:", err2)
		return false
	}
	return response.Result
}

func (b *CryptoBotApi) DeleteInvoice(id int) bool {
	proxyURL, err := url.Parse(b.proxyURL)
	if err != nil {
		fmt.Println("Ошибка разбора URL прокси:", err)
		return false
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	apiURL := "https://testnet-pay.crypt.bot/api/deleteInvoice"
	data := url.Values{}
	data.Set("invoice_id", strconv.Itoa(id))

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("Ошибка создания запроса:", err)
		return false
	}

	req.Header.Set("Crypto-Pay-API-Token", b.apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Ошибка выполнения запроса:", err)
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка чтения ответа:", err)
		return false
	}

	var response ResponseDelete
	err2 := json.Unmarshal(body, &response)
	if err2 != nil {
		fmt.Println("Error decoding JSON:", err2)
		return false
	}
	return response.Result
}

func (b *CryptoBotApi) GetMe() string {
	proxyURL, err := url.Parse(b.proxyURL)
	if err != nil {
		fmt.Println("Ошибка разбора URL прокси:", err)
		return ""
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	url := "https://testnet-pay.crypt.bot/api/getMe"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return ""
	}

	req.Header.Set("Crypto-Pay-API-Token", b.apiToken)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return ""
	}

	var response ResponseMe
	err2 := json.Unmarshal(body, &response)
	if err2 != nil {
		fmt.Println("Error decoding JSON:", err2)
		return ""
	}

	return response.Result.Name
}

func (b *CryptoBotApi) CreateInvoice(amount, asset, description string) (bool, string, int, string, string, string, string, string) {
	if ok, err := CheckNumber(amount); !ok {
		return false, err, 0, "", "", "", "", ""
	}

	proxyURL, err := url.Parse(b.proxyURL)
	if err != nil {
		fmt.Println("Ошибка разбора URL прокси:", err)
		return false, "", 0, "", "", "", "", ""
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	apiURL := "https://testnet-pay.crypt.bot/api/createInvoice"
	data := url.Values{}
	data.Set("amount", amount)
	data.Set("description", description)
	data.Set("asset", asset)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("Ошибка создания запроса:", err)
		return false, "", 0, "", "", "", "", ""
	}

	req.Header.Set("Crypto-Pay-API-Token", b.apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Ошибка выполнения запроса:", err)
		return false, "", 0, "","", "", "", ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка чтения ответа:", err)
		return false, "", 0, "", "", "", "", ""
	}

	var response ResponseInvoice
	err2 := json.Unmarshal(body, &response)
	if err2 != nil {
		fmt.Println("Error decoding JSON:", err2)
		return false, "", 0, "", "", "", "", ""
	}

	return true, "", response.Result.InvoiceID, response.Result.Amount, response.Result.Asset, response.Result.PayURL, response.Result.Description, response.Result.Status
}

func (b *CryptoBotApi) CreateCheck(amount, asset string) (bool, string, int, string, string, string, string){
	if ok, err := CheckNumber(amount); !ok {
		return false, err, 0, "", "", "", ""
	}

	proxyURL, err := url.Parse(b.proxyURL)
	if err != nil {
		fmt.Println("Ошибка разбора URL прокси:", err)
		return false, "", 0, "", "", "", ""
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	apiURL := "https://testnet-pay.crypt.bot/api/createCheck"
	data := url.Values{}
	data.Set("amount", amount)
	data.Set("asset", asset)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("Ошибка создания запроса:", err)
		return false, "", 0, "", "", "", ""
	}

	req.Header.Set("Crypto-Pay-API-Token", b.apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Ошибка выполнения запроса:", err)
		return false, "", 0, "", "", "", ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка чтения ответа:", err)
		return false, "", 0, "", "", "", ""
	}

	var response ResponseCheck
	err2 := json.Unmarshal(body, &response)
	if err2 != nil {
		fmt.Println("Error decoding JSON:", err2)
		return false, "", 0, "", "", "", ""
	}

	if response.Error != nil {
		return false, "", -1, "", "", "", ""
	}

    return true, "", response.Result.CheckID, response.Result.Amount, response.Result.Asset, response.Result.BotCheckURL, response.Result.Status
}

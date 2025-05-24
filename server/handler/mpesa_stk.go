package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/mediocregopher/radix/v4"

	gconfig "github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	nconfig "github.com/ortupik/wifigo/server/config"
	"github.com/ortupik/wifigo/server/database/model"
)

// MpesaConfig holds the general M-Pesa configuration loaded from Viper.
type MpesaConfig struct {
	Shortcode        string `mapstructure:"short_code"`
	Passkey          string `mapstructure:"passkey"`
	CallbackURL      string `mapstructure:"callback_url"`
	ConsumerKey      string `mapstructure:"consumer_key"`
	ConsumerSecret   string `mapstructure:"consumer_secret"`
	Environment      string `mapstructure:"environment"`
	TillNo           string `mapstructure:"till_no"`
	TransactionType  string `mapstructure:"transaction_type"`
	AccountReference string `mapstructure:"account_reference"`
	TransactionDesc  string `mapstructure:"transaction_desc"`
}

// NewMpesaStkHandler creates a new MpesaStkHandler with its dependencies.
func NewMpesaStkHandler() (*MpesaStkHandler, error) {
	mpesaConfig, err := LoadMpesaConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Mpesa config: %w", err)
	}
	return &MpesaStkHandler{
		mpesaConfig: mpesaConfig,
	}, nil
}

// MpesaStkHandler will contain dependencies for M-Pesa related logic.
type MpesaStkHandler struct {
	mpesaConfig *MpesaConfig
}

func LoadMpesaConfig() (*MpesaConfig, error) {
	var config MpesaConfig
	sub := nconfig.GetConfig().Sub("mpesa")
	fmt.Printf("mpesa config: %+v\n", sub)
	if sub == nil {
		return nil, fmt.Errorf("mpesa config not found")
	}
	if err := sub.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mpesa config: %w", err)
	}
	return &config, nil
}

// GetAccessToken retrieves the M-Pesa access token from Redis, or fetches a new one if expired.
func (h *MpesaStkHandler) GetAccessToken() (string, error) {
	redisClient := *gdatabase.GetRedis()
	rConnTTL := gconfig.GetConfig().Database.REDIS.Conn.ConnTTL
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rConnTTL)*time.Second)
	defer cancel()

	var token string
	err := redisClient.Do(ctx, radix.Cmd(&token, "GET", "mpesa:access_token"))
	if err == nil && token != "" {
		return token, nil
	}

	consumerKey := h.mpesaConfig.ConsumerKey
	consumerSecret := h.mpesaConfig.ConsumerSecret
	basicAuth := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + consumerSecret))

	req, err := http.NewRequest("GET", "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create M-Pesa access token request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+basicAuth)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch new M-Pesa access token: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in"`
	}

	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return "", fmt.Errorf("failed to decode M-Pesa access token response: %w", err)
	}

	expiresIn := time.Minute * 55
	err = redisClient.Do(ctx, radix.Cmd(nil, "SETEX", "mpesa:access_token", strconv.Itoa(int(expiresIn.Seconds())), tokenResp.AccessToken))
	if err != nil {
		return "", fmt.Errorf("failed to save M-Pesa access token to Redis with TTL: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// SendStkPush sends the STK push request to M-Pesa.
func (h *MpesaStkHandler) SendStkPush(phone, amount string) (map[string]interface{}, error) {
	accessToken, err := h.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	payload := map[string]interface{}{
		"BusinessShortCode": h.mpesaConfig.Shortcode,
		"Password":          generatePassword(h.mpesaConfig.Shortcode, h.mpesaConfig.Passkey, time.Now().Format("20060102150405")),
		"Timestamp":         time.Now().Format("20060102150405"),
		"TransactionType":   h.mpesaConfig.TransactionType,
		"Amount":            amount,
		"PartyA":            phone,
		"PartyB":            h.mpesaConfig.TillNo,
		"PhoneNumber":       formatPhoneNumber(phone),
		"CallBackURL":       h.mpesaConfig.CallbackURL,
		"AccountReference":  h.mpesaConfig.AccountReference,
		"TransactionDesc":   h.mpesaConfig.TransactionDesc,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal STK push payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.safaricom.co.ke/mpesa/stkpush/v1/processrequest", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create STK push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send STK push request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read STK push response body: %w", err)
	}

	var stkResponse map[string]interface{}
	if err := json.Unmarshal(responseBody, &stkResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal STK push response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || (stkResponse["errorCode"] != nil && stkResponse["errorCode"].(string) == "invalid_access_token") {
		fmt.Println("Safaricom rejected the access token. Refreshing...")
		if err := h.forceAccessTokenRefresh(); err != nil {
			fmt.Printf("Failed to force access token refresh: %v\n", err)
			return stkResponse, fmt.Errorf("authentication error and failed to refresh token: %w", err)
		}
		// Retry the STK push with the new token (recursive call or re-executing the logic)
		newAccessToken, err := h.GetAccessToken()
		if err != nil {
			return nil, fmt.Errorf("failed to get new access token after refresh: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+newAccessToken)
		resp, err = client.Do(req) // Retry the request
		// ... process the response of the retry ...
	}

	return stkResponse, nil
}

func (h *MpesaStkHandler) forceAccessTokenRefresh() error {
	redisClient := *gdatabase.GetRedis()
	rConnTTL := gconfig.GetConfig().Database.REDIS.Conn.ConnTTL
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rConnTTL)*time.Second)
	defer cancel()
	err := redisClient.Do(ctx, radix.Cmd(nil, "DEL", "mpesa:access_token"))
	return err
}

func (h *MpesaStkHandler) GetServicePlan(planId int) (model.ServicePlan, error) {
	var plan model.ServicePlan
	db := gdatabase.GetDB(gconfig.AppDB)
	result := db.First(&plan, "id = ?", planId)
	return plan, result.Error
}

func generatePassword(shortcode, passkey string, timestamp string) string {
	password := fmt.Sprintf("%s%s%s", shortcode, passkey, timestamp)
	encoded := base64.StdEncoding.EncodeToString([]byte(password))
	return encoded
}

func formatPhoneNumber(phone string) string {
	if len(phone) == 10 && phone[:1] == "0" {
		return "254" + phone[1:]
	} else if len(phone) == 9 && phone[:1] == "7" {
		return "254" + phone
	} else if len(phone) == 13 && phone[:4] == "+254" {
		return phone[1:] // Remove the '+'
	}
	return phone // Return the original if it doesn't match any expected format.  Consider logging an error.
}

package vertex

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/bytedance/gopkg/cache/asynccache"
	"github.com/golang-jwt/jwt/v5"

	"fmt"
	"time"
)

const metadataBaseURL = "http://metadata.google.internal/computeMetadata/v1"

type Credentials struct {
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	ClientID     string `json:"client_id"`
}

func IsADCKey(apiKey string) bool {
	switch strings.ToLower(strings.TrimSpace(apiKey)) {
	case "adc", "google_adc", "cloud_run_adc", "metadata":
		return true
	default:
		return false
	}
}

var Cache = asynccache.NewAsyncCache(asynccache.Options{
	RefreshDuration: time.Minute * 35,
	EnableExpire:    true,
	ExpireDuration:  time.Minute * 30,
	Fetcher: func(key string) (interface{}, error) {
		return nil, errors.New("not found")
	},
})

func getAccessToken(a *Adaptor, info *relaycommon.RelayInfo) (string, error) {
	var cacheKey string
	if info.ChannelIsMultiKey {
		cacheKey = fmt.Sprintf("access-token-%d-%d", info.ChannelId, info.ChannelMultiKeyIndex)
	} else {
		cacheKey = fmt.Sprintf("access-token-%d", info.ChannelId)
	}
	val, err := Cache.Get(cacheKey)
	if err == nil {
		return val.(string), nil
	}

	if a.UseADC {
		newToken, err := acquireMetadataAccessToken()
		if err != nil {
			return "", err
		}
		if err := Cache.SetDefault(cacheKey, newToken); err {
			return newToken, nil
		}
		return newToken, nil
	}

	signedJWT, err := createSignedJWT(a.AccountCredentials.ClientEmail, a.AccountCredentials.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create signed JWT: %w", err)
	}
	newToken, err := exchangeJwtForAccessToken(signedJWT, info)
	if err != nil {
		return "", fmt.Errorf("failed to exchange JWT for access token: %w", err)
	}
	if err := Cache.SetDefault(cacheKey, newToken); err {
		return newToken, nil
	}
	return newToken, nil
}

func GetADCProjectID() (string, error) {
	for _, envName := range []string{"GOOGLE_CLOUD_PROJECT", "GCP_PROJECT_ID", "GCLOUD_PROJECT"} {
		if projectID := strings.TrimSpace(os.Getenv(envName)); projectID != "" {
			return projectID, nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, metadataBaseURL+"/project/project-id", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := metadataHTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query metadata project id: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata project id request failed: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	projectID := strings.TrimSpace(string(body))
	if projectID == "" {
		return "", errors.New("metadata project id is empty")
	}
	return projectID, nil
}

func metadataHTTPClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

func acquireMetadataAccessToken() (string, error) {
	req, err := http.NewRequest(http.MethodGet, metadataBaseURL+"/instance/service-accounts/default/token", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := metadataHTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query metadata access token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata access token request failed: %s", resp.Status)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if accessToken, ok := result["access_token"].(string); ok && accessToken != "" {
		return accessToken, nil
	}
	return "", fmt.Errorf("metadata access token response missing access_token: %v", result)
}

func createSignedJWT(email, privateKeyPEM string) (string, error) {

	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "-----BEGIN PRIVATE KEY-----", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "-----END PRIVATE KEY-----", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "\r", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "\n", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "\\n", "")

	block, _ := pem.Decode([]byte("-----BEGIN PRIVATE KEY-----\n" + privateKeyPEM + "\n-----END PRIVATE KEY-----"))
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing the private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("not an RSA private key")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   email,
		"scope": "https://www.googleapis.com/auth/cloud-platform",
		"aud":   "https://www.googleapis.com/oauth2/v4/token",
		"exp":   now.Add(time.Minute * 35).Unix(),
		"iat":   now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(rsaPrivateKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func exchangeJwtForAccessToken(signedJWT string, info *relaycommon.RelayInfo) (string, error) {

	authURL := "https://www.googleapis.com/oauth2/v4/token"
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	data.Set("assertion", signedJWT)

	var client *http.Client
	var err error
	if info.ChannelSetting.Proxy != "" {
		client, err = service.NewProxyHttpClient(info.ChannelSetting.Proxy)
		if err != nil {
			return "", fmt.Errorf("new proxy http client failed: %w", err)
		}
	} else {
		client = service.GetHttpClient()
	}

	resp, err := client.PostForm(authURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if accessToken, ok := result["access_token"].(string); ok {
		return accessToken, nil
	}

	return "", fmt.Errorf("failed to get access token: %v", result)
}

func AcquireAccessToken(creds Credentials, proxy string) (string, error) {
	signedJWT, err := createSignedJWT(creds.ClientEmail, creds.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create signed JWT: %w", err)
	}
	return exchangeJwtForAccessTokenWithProxy(signedJWT, proxy)
}

func exchangeJwtForAccessTokenWithProxy(signedJWT string, proxy string) (string, error) {
	authURL := "https://www.googleapis.com/oauth2/v4/token"
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	data.Set("assertion", signedJWT)

	var client *http.Client
	var err error
	if proxy != "" {
		client, err = service.NewProxyHttpClient(proxy)
		if err != nil {
			return "", fmt.Errorf("new proxy http client failed: %w", err)
		}
	} else {
		client = service.GetHttpClient()
	}

	resp, err := client.PostForm(authURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if accessToken, ok := result["access_token"].(string); ok {
		return accessToken, nil
	}
	return "", fmt.Errorf("failed to get access token: %v", result)
}

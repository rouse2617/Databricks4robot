package argo

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains the Argo HTTP client configuration.
type Config struct {
	ServerURL          string
	Token              string
	InsecureSkipVerify bool
	CACertBase64       string
}

// ConfigFromEnv builds an Argo client config from environment variables.
func ConfigFromEnv() *Config {
	return &Config{
		ServerURL:          firstNonEmptyEnv("ARGO_SERVER_URL", "ARGO_BASE_URL"),
		Token:              firstNonEmptyEnv("ARGO_AUTH_TOKEN", "ARGO_TOKEN"),
		InsecureSkipVerify: parseBoolEnv("ARGO_INSECURE_SKIP_VERIFY"),
		CACertBase64:       strings.TrimSpace(os.Getenv("ARGO_CA_CERT_BASE64")),
	}
}

// NewClientFromConfig creates a Client from cfg.
func NewClientFromConfig(cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.InsecureSkipVerify || cfg.CACertBase64 != "" {
		tlsConfig := &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify} //nolint:gosec // Explicit caller config for private Argo endpoints.
		if cfg.CACertBase64 != "" {
			if raw, err := base64.StdEncoding.DecodeString(cfg.CACertBase64); err == nil {
				pool, _ := x509.SystemCertPool()
				if pool == nil {
					pool = x509.NewCertPool()
				}
				if pool.AppendCertsFromPEM(raw) {
					tlsConfig.RootCAs = pool
				}
			}
		}
		transport.TLSClientConfig = tlsConfig
	}

	return &Client{
		serverURL:  strings.TrimRight(strings.TrimSpace(cfg.ServerURL), "/"),
		token:      strings.TrimSpace(cfg.Token),
		httpClient: &http.Client{Transport: transport, Timeout: 60 * time.Second},
	}
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func parseBoolEnv(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

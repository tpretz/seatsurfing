package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Config struct {
	PublicListenAddr                    string
	PublicURL                           string
	FrontendURL                         string
	PostgresURL                         string
	JwtSigningKey                       string
	DisableUiProxy                      bool
	AdminUiBackend                      string
	BookingUiBackend                    string
	SMTPHost                            string
	SMTPPort                            int
	SMTPSenderAddress                   string
	SMTPStartTLS                        bool
	SMTPInsecureSkipVerify              bool
	SMTPAuth                            bool
	SMTPAuthUser                        string
	SMTPAuthPass                        string
	MockSendmail                        bool
	PrintConfig                         bool
	Development                         bool
	InitOrgName                         string
	InitOrgDomain                       string
	InitOrgUser                         string
	InitOrgPass                         string
	InitOrgCountry                      string
	InitOrgLanguage                     string
	OrgSignupEnabled                    bool
	OrgSignupDomain                     string
	OrgSignupAdmin                      string
	OrgSignupMaxUsers                   int
	OrgSignupDelete                     bool
	LoginProtectionMaxFails             int
	LoginProtectionSlidingWindowSeconds int
	LoginProtectionBanMinutes           int
	AWSAccessKey                        string
	AWSSecretAccessKey                  string
	AWSSessionToken                     string
	AWSSNSTopic                         string
	AWSRegion                           string
}

var _configInstance *Config
var _configOnce sync.Once

func GetConfig() *Config {
	_configOnce.Do(func() {
		_configInstance = &Config{}
		_configInstance.ReadConfig()
	})
	return _configInstance
}

func (c *Config) ReadConfig() {
	log.Println("Reading config...")
	c.Development = (c.getEnv("DEV", "0") == "1")
	c.PublicListenAddr = c.getEnv("PUBLIC_LISTEN_ADDR", "0.0.0.0:8080")
	c.PublicURL = strings.TrimSuffix(c.getEnv("PUBLIC_URL", "http://localhost:8080"), "/") + "/"
	if c.Development {
		c.FrontendURL = c.getEnv("FRONTEND_URL", "http://localhost:3000")
	} else {
		c.FrontendURL = c.getEnv("FRONTEND_URL", "http://localhost:8080")
	}
	c.FrontendURL = strings.TrimSuffix(c.FrontendURL, "/") + "/"
	c.DisableUiProxy = (c.getEnv("DISABLE_UI_PROXY", "0") == "1")
	c.AdminUiBackend = c.getEnv("ADMIN_UI_BACKEND", "localhost:3000")
	c.BookingUiBackend = c.getEnv("BOOKING_UI_BACKEND", "localhost:3001")
	c.PostgresURL = c.getEnv("POSTGRES_URL", "postgres://postgres:root@localhost/seatsurfing?sslmode=disable")
	c.JwtSigningKey = c.getEnv("JWT_SIGNING_KEY", "cX32hEwZDCLZ6bCR")
	c.SMTPHost = c.getEnv("SMTP_HOST", "127.0.0.1")
	c.SMTPPort = c.getEnvInt("SMTP_PORT", 25)
	c.SMTPStartTLS = (c.getEnv("SMTP_START_TLS", "0") == "1")
	c.SMTPInsecureSkipVerify = (c.getEnv("SMTP_INSECURE_SKIP_VERIFY", "0") == "1")
	c.SMTPAuth = (c.getEnv("SMTP_AUTH", "0") == "1")
	c.SMTPAuthUser = c.getEnv("SMTP_AUTH_USER", "")
	c.SMTPAuthPass = c.getEnv("SMTP_AUTH_PASS", "")
	c.SMTPSenderAddress = c.getEnv("SMTP_SENDER_ADDRESS", "no-reply@seatsurfing.local")
	c.MockSendmail = (c.getEnv("MOCK_SENDMAIL", "0") == "1")
	c.PrintConfig = (c.getEnv("PRINT_CONFIG", "0") == "1")
	c.InitOrgName = c.getEnv("INIT_ORG_NAME", "Sample Company")
	c.InitOrgDomain = c.getEnv("INIT_ORG_DOMAIN", "seatsurfing.local")
	c.InitOrgUser = c.getEnv("INIT_ORG_USER", "admin")
	c.InitOrgPass = c.getEnv("INIT_ORG_PASS", "12345678")
	c.InitOrgCountry = c.getEnv("INIT_ORG_COUNTRY", "DE")
	c.InitOrgLanguage = c.getEnv("INIT_ORG_LANGUAGE", "de")
	c.OrgSignupEnabled = (c.getEnv("ORG_SIGNUP_ENABLED", "0") == "1")
	c.OrgSignupDomain = c.getEnv("ORG_SIGNUP_DOMAIN", ".on.seatsurfing.local")
	c.OrgSignupAdmin = c.getEnv("ORG_SIGNUP_ADMIN", "admin")
	c.OrgSignupMaxUsers = c.getEnvInt("ORG_SIGNUP_MAX_USERS", 10)
	c.OrgSignupDelete = (c.getEnv("ORG_SIGNUP_DELETE", "0") == "1")
	c.LoginProtectionMaxFails = c.getEnvInt("LOGIN_PROTECTION_MAX_FAILS", 10)
	c.LoginProtectionSlidingWindowSeconds = c.getEnvInt("LOGIN_PROTECTION_SLIDING_WINDOW_SECONDS", 600)
	c.LoginProtectionBanMinutes = c.getEnvInt("LOGIN_PROTECTION_BAN_MINUTES", 5)
	c.AWSAccessKey = c.getEnv("AWS_ACCESS_KEY_ID", "")
	c.AWSSecretAccessKey = c.getEnv("AWS_SECRET_ACCESS_KEY", "")
	c.AWSSessionToken = c.getEnv("AWS_SESSION_TOKEN", "")
	c.AWSSNSTopic = c.getEnv("AWS_SNS_TOPIC", "")
	c.AWSRegion = c.getEnv("AWS_REGION", "")
}

func (c *Config) Print() {
	s, _ := json.MarshalIndent(c, "", "\t")
	log.Println("Using config:\n" + string(s))
}

func (c *Config) getEnv(key, defaultValue string) string {
	res := os.Getenv(key)
	if res == "" {
		return defaultValue
	}
	return res
}

func (c *Config) getEnvInt(key string, defaultValue int) int {
	val, err := strconv.Atoi(c.getEnv(key, strconv.Itoa(defaultValue)))
	if err != nil {
		log.Fatal("Could not parse " + key + " to int")
	}
	return val
}

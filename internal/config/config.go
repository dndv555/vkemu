package config

import (
	"flag"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultAppID     = "2274003"
	DefaultAppSecret = "5ZNPAwJZ6NBW96BKDaCQ5ZCZpR4BkLyhAzpmpYZWCFCTuDRRbz"
)

type Config struct {
	Addr             string
	WebAddr          string
	DBPath           string
	UploadDir        string
	BaseURL          string
	LongPollHost     string
	RequireSignature bool
	AppID            string
	AppSecret        string
	LongPollWait     int
	RateLimit        float64
	RateBurst        float64
	MaxConns         int
	TrustProxy       bool
}

func Load() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.Addr, "addr", env("VKEMU_ADDR", ":8080"), "адрес прослушивания API")
	flag.StringVar(&cfg.WebAddr, "web-addr", env("VKEMU_WEB_ADDR", ":80"), "адрес веб-версии")
	flag.StringVar(&cfg.DBPath, "db", env("VKEMU_DB", "data/vkemu.db"), "путь к базе SQLite")
	flag.StringVar(&cfg.UploadDir, "uploads", env("VKEMU_UPLOADS", "data/uploads"), "каталог загруженных файлов")
	flag.StringVar(&cfg.BaseURL, "base-url", env("VKEMU_BASE_URL", ""), "внешний адрес сервера, например http://10.0.2.2:8080")
	flag.StringVar(&cfg.LongPollHost, "longpoll-host", env("VKEMU_LONGPOLL_HOST", ""), "хост long poll в формате host:port/longpoll")
	flag.BoolVar(&cfg.RequireSignature, "require-signature", envBool("VKEMU_REQUIRE_SIGNATURE", false), "проверять sig запросов")
	flag.StringVar(&cfg.AppID, "app-id", env("VKEMU_APP_ID", DefaultAppID), "ожидаемый api_id")
	flag.StringVar(&cfg.AppSecret, "app-secret", env("VKEMU_APP_SECRET", DefaultAppSecret), "секрет secure-авторизации")
	flag.IntVar(&cfg.LongPollWait, "longpoll-wait", envInt("VKEMU_LONGPOLL_WAIT", 25), "ожидание long poll, секунды")
	flag.Float64Var(&cfg.RateLimit, "rate-limit", envFloat("VKEMU_RATE_LIMIT", 50), "запросов в секунду на один IP (0 — выключить)")
	flag.Float64Var(&cfg.RateBurst, "rate-burst", envFloat("VKEMU_RATE_BURST", 100), "максимальный всплеск запросов на один IP")
	flag.IntVar(&cfg.MaxConns, "max-conns", envInt("VKEMU_MAX_CONNS", 512), "максимум одновременных обработок (0 — без лимита)")
	flag.BoolVar(&cfg.TrustProxy, "trust-proxy", envBool("VKEMU_TRUST_PROXY", false), "использовать X-Forwarded-For для определения IP")
	flag.Parse()

	if cfg.BaseURL == "" {
		host := cfg.Addr
		if strings.HasPrefix(host, ":") {
			host = "127.0.0.1" + host
		}
		cfg.BaseURL = "http://" + host
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.LongPollHost == "" {
		host := strings.TrimPrefix(cfg.BaseURL, "http://")
		host = strings.TrimPrefix(host, "https://")
		cfg.LongPollHost = host + "/longpoll"
	}
	return cfg
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Bootstrap struct {
	Environment  string
	DatabaseURL  string
	MasterKEK    []byte
	HTTPAddr     string
	InternalAddr string
}

func Load(getenv func(string) string) (Bootstrap, error) {
	c := Bootstrap{Environment: getenv("APP_ENV"), DatabaseURL: getenv("DATABASE_URL"), HTTPAddr: getenv("HTTP_ADDR"), InternalAddr: getenv("INTERNAL_ADDR")}
	if c.Environment != "development" && c.Environment != "test" {
		return c, errors.New("APP_ENV must be development or test; production gates are not complete")
	}
	u, err := url.Parse(c.DatabaseURL)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" || u.User == nil || u.Path == "" {
		return c, errors.New("DATABASE_URL is invalid")
	}
	c.MasterKEK, err = base64.StdEncoding.Strict().DecodeString(getenv("MASTER_KEK"))
	if err != nil || len(c.MasterKEK) != 32 {
		return c, errors.New("MASTER_KEK must be base64 of 32 bytes")
	}
	if c.HTTPAddr == "" {
		c.HTTPAddr = "127.0.0.1:8080"
	}
	if c.InternalAddr == "" {
		c.InternalAddr = "127.0.0.1:9090"
	}
	for _, addr := range []string{c.HTTPAddr, c.InternalAddr} {
		host, port, err := net.SplitHostPort(addr)
		n, e := strconv.Atoi(port)
		if err != nil || e != nil || n < 1 || n > 65535 || (host != "" && net.ParseIP(host) == nil && host != "localhost") {
			return c, errors.New("invalid listener address")
		}
	}
	if c.HTTPAddr == c.InternalAddr {
		return c, errors.New("public and internal listeners must differ")
	}
	return c, nil
}

type Settings struct {
	BannerRU              string   `json:"bannerRu,omitempty"`
	BannerEN              string   `json:"bannerEn,omitempty"`
	FileTypeMode          string   `json:"fileTypeMode,omitempty"`
	FileTypes             []string `json:"fileTypes,omitempty"`
	PublicOrigin          string   `json:"publicOrigin"`
	SessionTTLSeconds     int64    `json:"sessionTtlSeconds"`
	Version               int      `json:"version"`
	TransferMaxTTLSeconds int64    `json:"transferMaxTtlSeconds"`
	MaxFiles              int      `json:"maxFiles"`
	MaxPathBytes          int      `json:"maxPathBytes"`
}

func ParseSettings(raw []byte) (Settings, error) {
	var s Settings
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if err := d.Decode(&s); err != nil {
		return s, errors.New("invalid settings schema")
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return s, errors.New("trailing settings data")
	}
	if s.PublicOrigin == "" {
		s.PublicOrigin = "http://localhost:8090"
	}
	if s.SessionTTLSeconds == 0 {
		s.SessionTTLSeconds = 43200
	}
	origin, originErr := url.Parse(s.PublicOrigin)
	if originErr != nil || origin.User != nil || origin.Host == "" || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" || (origin.Scheme != "https" && !(origin.Scheme == "http" && (origin.Hostname() == "localhost" || (net.ParseIP(origin.Hostname()) != nil && net.ParseIP(origin.Hostname()).IsLoopback())))) || s.SessionTTLSeconds < 300 || s.SessionTTLSeconds > 604800 {
		return s, errors.New("invalid origin or session lifetime")
	}
	if len(s.BannerRU) > 2000 || len(s.BannerEN) > 2000 {
		return s, errors.New("banner too long")
	}
	if s.Version != 1 || s.TransferMaxTTLSeconds < 1 || s.TransferMaxTTLSeconds > 31536000 || s.MaxFiles < 1 || s.MaxFiles > 100000 || s.MaxPathBytes < 1 || s.MaxPathBytes > 4096 {
		return s, errors.New("settings outside supported bounds")
	}
	if s.FileTypeMode != "" && s.FileTypeMode != "disabled" && s.FileTypeMode != "allow" && s.FileTypeMode != "deny" {
		return s, errors.New("invalid file policy")
	}
	if len(s.FileTypes) > 100 || (s.FileTypeMode == "allow" && len(s.FileTypes) == 0) {
		return s, errors.New("invalid file policy")
	}
	for _, v := range s.FileTypes {
		if !strings.Contains(v, "/") || strings.ContainsAny(v, " \r\n;") {
			return s, errors.New("invalid file type")
		}
	}
	return s, nil
}

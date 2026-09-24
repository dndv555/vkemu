package params

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type Params map[string]string

func New() Params {
	return Params{}
}

func FromValues(values url.Values) Params {
	p := make(Params, len(values))
	for k, v := range values {
		if len(v) > 0 {
			p[k] = v[0]
		}
	}
	return p
}

func FromAny(src map[string]any) Params {
	p := make(Params, len(src))
	for k, v := range src {
		p[k] = scalar(v)
	}
	return p
}

func (p Params) Clone() Params {
	out := make(Params, len(p))
	for k, v := range p {
		out[k] = v
	}
	return out
}

func (p Params) Get(key string) string {
	return p[key]
}

func (p Params) Has(key string) bool {
	_, ok := p[key]
	return ok
}

func (p Params) String(key, def string) string {
	if v, ok := p[key]; ok && v != "" {
		return v
	}
	return def
}

func (p Params) Int(key string, def int) int {
	v, ok := p[key]
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return def
		}
		return int(f)
	}
	return n
}

func (p Params) Int64(key string, def int64) int64 {
	return int64(p.Int(key, int(def)))
}

func (p Params) Float(key string, def float64) float64 {
	v, ok := p[key]
	if !ok || v == "" {
		return def
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return def
	}
	return f
}

func (p Params) Bool(key string, def bool) bool {
	v, ok := p[key]
	if !ok || v == "" {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	}
	return def
}

func (p Params) List(key string) []string {
	v := strings.TrimSpace(p[key])
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (p Params) IntList(key string) []int {
	parts := p.List(key)
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err == nil {
			out = append(out, n)
		}
	}
	return out
}

func (p Params) Keys() []string {
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (p Params) Signature(uid int, secret string) string {
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(uid))
	for _, key := range p.Keys() {
		if key == "sig" || key == "sid" {
			continue
		}
		sb.WriteString(key)
		sb.WriteString("=")
		sb.WriteString(p[key])
	}
	sb.WriteString(secret)
	return MD5(sb.String())
}

func MD5(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func scalar(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "1"
		}
		return "0"
	case float64:
		return formatNumber(x)
	case float32:
		return formatNumber(float64(x))
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case []any:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			parts = append(parts, scalar(item))
		}
		return strings.Join(parts, ",")
	case []string:
		return strings.Join(x, ",")
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

func formatNumber(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

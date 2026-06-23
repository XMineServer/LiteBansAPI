package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"xmine/litebans-api/internal/domain"
)

// idModulus is a prime > any realistic auto-increment id (2^31 - 1), making encode/decode a true
// bijection over [0, idModulus-1] with no collisions within a punishment type.
const idModulus int64 = 2147483647

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// IDObfuscator reversibly maps internal numeric punishment ids to opaque tokens (TOR 7.3),
// using a deployment secret so the mapping can't be derived without it.
type IDObfuscator struct {
	enabled bool
	secret  string
}

func NewIDObfuscator(enabled bool, secret string) *IDObfuscator {
	return &IDObfuscator{enabled: enabled, secret: secret}
}

func (o *IDObfuscator) Enabled() bool {
	return o.enabled
}

func typeTag(t domain.PunishmentType) byte {
	switch t {
	case domain.TypeBan:
		return 'b'
	case domain.TypeMute:
		return 'm'
	case domain.TypeWarning:
		return 'w'
	case domain.TypeKick:
		return 'k'
	default:
		return '?'
	}
}

func tagType(tag byte) (domain.PunishmentType, bool) {
	switch tag {
	case 'b':
		return domain.TypeBan, true
	case 'm':
		return domain.TypeMute, true
	case 'w':
		return domain.TypeWarning, true
	case 'k':
		return domain.TypeKick, true
	default:
		return "", false
	}
}

// keyPair derives a deterministic (mult, add) pair for a punishment type from the deployment secret.
func (o *IDObfuscator) keyPair(t domain.PunishmentType) (mult, add int64) {
	mac := hmac.New(sha256.New, []byte(o.secret))
	mac.Write([]byte("litebans-api-id-obfuscation:" + string(t)))
	sum := mac.Sum(nil)
	mult = int64(binary.BigEndian.Uint32(sum[0:4])) % (idModulus - 1)
	if mult < 0 {
		mult += idModulus - 1
	}
	mult++ // ensure mult in [1, idModulus-1], coprime to prime idModulus
	add = int64(binary.BigEndian.Uint32(sum[4:8])) % idModulus
	if add < 0 {
		add += idModulus
	}
	return mult, add
}

// modInverse returns x such that (a*x) % m == 1, via the extended Euclidean algorithm.
func modInverse(a, m int64) int64 {
	g, x, _ := extendedGCD(a, m)
	if g != 1 {
		panic("modInverse: a and m are not coprime")
	}
	x %= m
	if x < 0 {
		x += m
	}
	return x
}

func extendedGCD(a, b int64) (g, x, y int64) {
	if b == 0 {
		return a, 1, 0
	}
	g, x1, y1 := extendedGCD(b, a%b)
	return g, y1, x1 - (a/b)*y1
}

// Encode converts an internal id into an opaque, type-tagged token.
func (o *IDObfuscator) Encode(t domain.PunishmentType, id int64) string {
	mult, add := o.keyPair(t)
	x := mulMod(id, mult, idModulus) + add
	x %= idModulus
	return string(typeTag(t)) + base62Encode(x)
}

// Decode reverses Encode, returning an error if the token's type tag doesn't match t or the token is malformed.
func (o *IDObfuscator) Decode(t domain.PunishmentType, token string) (int64, error) {
	if len(token) < 2 {
		return 0, fmt.Errorf("invalid id token")
	}
	tag, ok := tagType(token[0])
	if !ok || tag != t {
		return 0, fmt.Errorf("id token does not match punishment type")
	}
	x, err := base62Decode(token[1:])
	if err != nil {
		return 0, fmt.Errorf("invalid id token: %w", err)
	}
	mult, add := o.keyPair(t)
	inv := modInverse(mult, idModulus)
	diff := (x - add) % idModulus
	if diff < 0 {
		diff += idModulus
	}
	return mulMod(diff, inv, idModulus), nil
}

// mulMod computes (a*b) % m. Safe here because a, b < idModulus < 2^31, so a*b fits in int64.
func mulMod(a, b, m int64) int64 {
	return (a % m) * (b % m) % m
}

func base62Encode(n int64) string {
	if n == 0 {
		return "0"
	}
	var sb strings.Builder
	for n > 0 {
		sb.WriteByte(base62Alphabet[n%62])
		n /= 62
	}
	s := sb.String()
	return reverseString(s)
}

func base62Decode(s string) (int64, error) {
	var n int64
	for _, c := range s {
		idx := strings.IndexRune(base62Alphabet, c)
		if idx < 0 {
			return 0, fmt.Errorf("invalid base62 character %q", c)
		}
		n = n*62 + int64(idx)
	}
	return n, nil
}

func reverseString(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

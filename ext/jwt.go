package ext

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// signHS256 signs the given JWT payload using the HMAC-SHA256 algorithm and
// returns the compact JWS serialization. Mercure accepts HS256 tokens signed
// with the keys configured via the `publisher_jwt`/`subscriber_jwt` Caddyfile
// options.
func signHS256(secret string, payload []byte) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(header))
	mac.Write([]byte("."))
	mac.Write([]byte(body))

	return header + "." + body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func claimsJSON(mercure map[string]any, issuedAt time.Time, ttl time.Duration) []byte {
	claims := map[string]any{
		"iat":     issuedAt.Unix(),
		"exp":     issuedAt.Add(ttl).Unix(),
		"mercure": mercure,
	}

	encoded, err := json.Marshal(claims)
	if err != nil {
		// Built from fixed keys; cannot fail.
		panic(fmt.Sprintf("marshal mercure JWT claims: %v", err))
	}
	return encoded
}

// PublishJWT mints a publisher token allowed to update every topic.
func PublishJWT(secret string, ttl time.Duration, now time.Time) string {
	return signHS256(secret, claimsJSON(map[string]any{
		"publish": []string{"*"},
	}, now, ttl))
}

// SubscriptionsJWT mints a subscriber token that grants access to the
// subscription listing API.
func SubscriptionsJWT(secret string, ttl time.Duration, now time.Time) string {
	return signHS256(secret, claimsJSON(map[string]any{
		"subscribe": []string{"*"},
		"payload":   map[string]any{"subscriptions": true},
	}, now, ttl))
}

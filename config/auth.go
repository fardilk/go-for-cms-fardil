package config

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// JWTSecret returns the signing key. It used to be the literal
// "your_secret_key" hardcoded in three handlers, which let anyone forge a
// token, so an unset secret is fatal rather than silently insecure.
func JWTSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		if AppEnv() == "production" {
			log.Fatal("JWT_SECRET must be set in production")
		}
		log.Println("WARNING: JWT_SECRET is unset, falling back to a development key")
		s = "dev-only-insecure-key"
	}
	return []byte(s)
}

// AppEnv is "production" on the server and anything else locally.
func AppEnv() string {
	if e := os.Getenv("APP_ENV"); e != "" {
		return e
	}
	return "development"
}

// IsProduction reports whether cookies must be Secure and seeding must stay off.
func IsProduction() bool {
	return AppEnv() == "production"
}

// Port is the listen address, previously hardcoded to :8000 despite the comment
// claiming it came from .env.
func Port() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	return ":8000"
}

// AllowedOrigins lists the browser origins permitted by CORS. The default is
// the local CMS dev server; production sets CORS_ORIGINS.
func AllowedOrigins() []string {
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		var out []string
		for _, o := range strings.Split(v, ",") {
			if o = strings.TrimSpace(o); o != "" {
				out = append(out, o)
			}
		}
		return out
	}
	return []string{"http://localhost:5173"}
}

// UploadDir is where uploaded media is written. It must point at a mounted
// volume in production, or uploads vanish when the container is recreated.
func UploadDir() string {
	if d := os.Getenv("UPLOAD_DIR"); d != "" {
		return d
	}
	return "./static/images"
}

// CookieDomain scopes the session cookie so the CMS and the API can share it
// across subdomains. Empty means host-only, which is right for local dev.
func CookieDomain() string {
	return os.Getenv("COOKIE_DOMAIN")
}

// MediaQuotaBytes caps the total size of stored uploads. The upload directory
// sits on a fixed-size volume, so this exists to fail with a clear message
// before the filesystem fails with ENOSPC. Zero disables the check.
//
// ponytail: sums the recorded sizes rather than calling statfs, which keeps the
// code portable; it undercounts anything written to the volume outside the API.
func MediaQuotaBytes() int64 {
	v := os.Getenv("MEDIA_QUOTA_BYTES")
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		log.Printf("WARNING: ignoring invalid MEDIA_QUOTA_BYTES=%q", v)
		return 0
	}
	return n
}

// CookieSameSite controls the session cookie's SameSite attribute. The panel is
// served from the same origin as the API, so Lax is both sufficient and safer
// than None; set COOKIE_SAMESITE=none only if the CMS moves to its own domain.
func CookieSameSite() http.SameSite {
	switch strings.ToLower(os.Getenv("COOKIE_SAMESITE")) {
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return http.SameSiteLaxMode
	}
}

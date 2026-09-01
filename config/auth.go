package config

import (
	"log"
	"os"
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

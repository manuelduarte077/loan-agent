package http

import (
	"log"
	"net"
	"net/http"
	"strings"
)

// extractClientIP extrae la IP del cliente de la request, considerando proxies
func extractClientIP(r *http.Request) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" && net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		realIP = strings.TrimSpace(realIP)
		if net.ParseIP(realIP) != nil {
			return realIP
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		if net.ParseIP(r.RemoteAddr) != nil {
			return r.RemoteAddr
		}
		log.Printf("Warning: failed to parse RemoteAddr '%s': %v, using as-is", r.RemoteAddr, err)
		return r.RemoteAddr
	}

	if ip == "" {
		log.Printf("Warning: empty IP from RemoteAddr '%s', using RemoteAddr", r.RemoteAddr)
		return r.RemoteAddr
	}

	if net.ParseIP(ip) == nil {
		log.Printf("Warning: invalid IP '%s' from RemoteAddr '%s', using RemoteAddr", ip, r.RemoteAddr)
		return r.RemoteAddr
	}

	return ip
}

func RateLimitMiddleware(
	limiter *RateLimiter,
	next http.Handler,
) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractClientIP(r)

		if !limiter.Allow(ip) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

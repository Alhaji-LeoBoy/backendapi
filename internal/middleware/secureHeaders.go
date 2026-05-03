package middleware

import "net/http"

// SecureHeaders adds security-related HTTP response headers to every request.
// Place this early in the middleware chain, before auth and routing.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent browsers from MIME-sniffing the content type.
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Block the page from being embedded in an iframe (clickjacking protection).
		w.Header().Set("X-Frame-Options", "DENY")

		// Force HTTPS for 2 years; include subdomains.
		// Only effective over TLS — safe to set always, browsers ignore it over HTTP.
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")

		// Restrict what the browser is allowed to load.
		// Tighten default-src if you serve static assets from known origins.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self'; "+
				"img-src 'self' data:; "+
				"font-src 'self'; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'",
		)

		// Stop sending the Referer header when navigating to a different origin.
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Disable access to browser features/APIs not needed by this API.
		w.Header().Set("Permissions-Policy",
			"geolocation=(), "+
				"camera=(), "+
				"microphone=(), "+
				"payment=(), "+
				"usb=()",
		)

		// Tell the browser not to cache responses that may contain sensitive data.
		// Override per-route for public cacheable resources (e.g. /health).
		w.Header().Set("Cache-Control", "no-store")

		// Legacy XSS filter for older browsers (IE/Edge pre-Chromium).
		// Mostly redundant with CSP, but harmless to include.
		w.Header().Set("X-XSS-Protection", "0")

		next.ServeHTTP(w, r)
	})
}

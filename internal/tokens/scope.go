package tokens

const (
	// Token Scopes
	ScopeAccess  = "access"  // Short-lived (15 min) - for API access
	ScopeRefresh = "refresh" // Long-lived (7 days) - for getting new access tokens
	ScopeAPI     = "api"     // Permanent - for machine-to-machine (AI model access)

	// One-time Token Scopes
	ScopeActivation    = "activation"     // Email verification (24 hours)
	ScopePasswordReset = "password_reset" // Forgot password (1 hour)
	ScopeEmailChange   = "email_change"   // Email change verification (1 hour)

	// Permission Scopes (for API keys)
	ScopeRead  = "read"  // Read-only access
	ScopeWrite = "write" // Read and write access
	ScopeAdmin = "admin" // Full access including user management
)

package middleware

import (
	auditservice "api-server/internal/app/services/audit"
	authservice "api-server/internal/app/services/auth"
	"api-server/internal/constants"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	authService *authservice.AuthService
	// cookieSecure mirrors Secure on the session cookie: on in production
	// (HTTPS terminated at nginx), off in dev so localhost keeps working.
	cookieSecure bool
}

// NewAuthMiddleware constructs the middleware.
func NewAuthMiddleware(authService *authservice.AuthService, cookieSecure bool) *AuthMiddleware {
	return &AuthMiddleware{
		authService:  authService,
		cookieSecure: cookieSecure,
	}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractSessionToken(c)
		if token == "" {
			response.Unauthorized(c, "Xin đăng nhập để tiếp tục")
			c.Abort()
			return
		}

		claims, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			response.Unauthorized(c, "Invalid token")
			c.Abort()
			return
		}

		// Migration auto-upgrade (PAYR-37B item 4): the caller authenticated
		// via a legacy Bearer header and has no session cookie yet. Mint one
		// for the token's remaining lifetime so a cached old-SPA session
		// converts to cookie auth on its first post-deploy request — no
		// forced re-login during the localStorage → HttpOnly rollout.
		if authenticatedViaBearerHeader(c) {
			if _, err := c.Cookie(AuthCookieName); err != nil && claims.ExpiresAt != nil {
				if remaining := claims.ExpiresAt.Sub(clock.Now()); remaining > 0 {
					SetAuthCookie(c, token, remaining, m.cookieSecure)
				}
			}
		}

		// Set user info in context
		c.Set(constants.CtxUserID, claims.UserID)
		c.Set("username", claims.Username)
		c.Set(constants.CtxUserRole, claims.Role)
		c.Set("token_jti", claims.ID)
		// otp_verified: propagated from the JWT claim so Authorize() can gate
		// privileged routes on completion of the email-OTP second factor (RT-C1).
		c.Set("otp_verified", claims.OTPVerified)
		// Set tenant_id for tenant isolation middleware (using user_id as tenant identifier)
		c.Set("tenant_id", claims.UserID)

		// Add user information to audit context using shared helper
		ctx := auditservice.WithAuditContext(c.Request.Context(), claims.UserID, claims.Fullname)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractSessionToken(c)
		if token == "" {
			c.Next()
			return
		}

		if claims, err := m.authService.ValidateToken(c.Request.Context(), token); err == nil {
			c.Set(constants.CtxUserID, claims.UserID)
			c.Set("username", claims.Username)
			c.Set(constants.CtxUserRole, claims.Role)
			c.Set("token_jti", claims.ID)
			c.Set("otp_verified", claims.OTPVerified)
			// Set tenant_id for tenant isolation middleware (using user_id as tenant identifier)
			c.Set("tenant_id", claims.UserID)

			// Add user information to audit context using shared helper
			ctx := auditservice.WithAuditContext(c.Request.Context(), claims.UserID, claims.Fullname)
			c.Request = c.Request.WithContext(ctx)
		}

		c.Next()
	}
}

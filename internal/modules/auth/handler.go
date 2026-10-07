package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for authentication and session management.
type Handler struct {
	service Service
}

// NewHandler constructs a new auth HTTP handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// Login handles user authentication via email and password.
// POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: email and password are required"})
		return
	}

	cleanEmail := strings.TrimSpace(req.Email)
	if cleanEmail == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password cannot be empty"})
		return
	}

	pair, user, err := h.service.Login(c.Request.Context(), cleanEmail, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUserInactive) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error during login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  pair.AccessToken,
		"refreshToken": pair.RefreshToken,
		"expiresIn":    pair.ExpiresIn,
		"user":         user,
	})
}

// Refresh handles token rotation using a valid refresh token.
// POST /api/v1/auth/refresh
func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	_ = c.ShouldBindJSON(&req)

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		refreshToken = c.GetHeader("X-Refresh-Token")
	}

	if strings.TrimSpace(refreshToken) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token is required"})
		return
	}

	pair, user, err := h.service.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  pair.AccessToken,
		"refreshToken": pair.RefreshToken,
		"expiresIn":    pair.ExpiresIn,
		"user":         user,
	})
}

// Logout revokes the provided refresh token session.
// POST /api/v1/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	var req refreshRequest
	_ = c.ShouldBindJSON(&req)

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		refreshToken = c.GetHeader("X-Refresh-Token")
	}

	if refreshToken != "" {
		_ = h.service.Logout(c.Request.Context(), refreshToken)
	}

	c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

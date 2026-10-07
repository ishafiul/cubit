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

type setupRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Status returns whether the cluster has completed initial setup.
// GET /api/v1/auth/status
func (h *Handler) Status(c *gin.Context) {
	status, err := h.service.GetStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check setup status"})
		return
	}
	c.JSON(http.StatusOK, status)
}

// Setup provisions the root administrator and initializes the cluster.
// POST /api/v1/auth/setup
func (h *Handler) Setup(c *gin.Context) {
	var req setupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: name, email, and password are required"})
		return
	}

	cleanName := strings.TrimSpace(req.Name)
	cleanEmail := strings.TrimSpace(req.Email)
	if cleanName == "" || cleanEmail == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, email, and password cannot be empty"})
		return
	}

	pair, user, err := h.service.Setup(c.Request.Context(), cleanName, cleanEmail, req.Password)
	if err != nil {
		if errors.Is(err, ErrSetupAlreadyCompleted) {
			c.JSON(http.StatusConflict, gin.H{"error": "setup has already been completed"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error during setup"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"accessToken":  pair.AccessToken,
		"refreshToken": pair.RefreshToken,
		"expiresIn":    pair.ExpiresIn,
		"user":         user,
	})
}

type createRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type updateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// Permissions returns the list of all platform permissions.
// GET /api/v1/permissions
func (h *Handler) Permissions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"permissions": h.service.ListPermissions(),
	})
}

// ListRoles returns all roles.
// GET /api/v1/roles
func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.service.ListRoles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list roles"})
		return
	}
	c.JSON(http.StatusOK, roles)
}

// GetRole returns a role by ID.
// GET /api/v1/roles/:id
func (h *Handler) GetRole(c *gin.Context) {
	id := c.Param("id")
	role, err := h.service.GetRole(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get role"})
		return
	}
	c.JSON(http.StatusOK, role)
}

// CreateRole creates a new custom role.
// POST /api/v1/roles
func (h *Handler) CreateRole(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: role name is required"})
		return
	}

	cleanName := strings.TrimSpace(req.Name)
	if cleanName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role name cannot be empty"})
		return
	}

	role, err := h.service.CreateRole(c.Request.Context(), cleanName, req.Description, req.Permissions)
	if err != nil {
		if errors.Is(err, ErrRoleAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "role with this name already exists"})
			return
		}
		if errors.Is(err, ErrInvalidRoleName) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role name cannot be empty"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create role"})
		return
	}

	c.JSON(http.StatusCreated, role)
}

// UpdateRole updates an existing custom role.
// PUT /api/v1/roles/:id
func (h *Handler) UpdateRole(c *gin.Context) {
	id := c.Param("id")
	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: role name is required"})
		return
	}

	cleanName := strings.TrimSpace(req.Name)
	if cleanName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role name cannot be empty"})
		return
	}

	role, err := h.service.UpdateRole(c.Request.Context(), id, cleanName, req.Description, req.Permissions)
	if err != nil {
		if errors.Is(err, ErrCannotModifySystemRole) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot modify system roles"})
			return
		}
		if errors.Is(err, ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
			return
		}
		if errors.Is(err, ErrRoleAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "role with this name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
		return
	}

	c.JSON(http.StatusOK, role)
}

// DeleteRole removes an existing custom role.
// DELETE /api/v1/roles/:id
func (h *Handler) DeleteRole(c *gin.Context) {
	id := c.Param("id")
	err := h.service.DeleteRole(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCannotDeleteSystemRole) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete system roles"})
			return
		}
		if errors.Is(err, ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}


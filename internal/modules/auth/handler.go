package auth

import (
	"net/http"

	"nexora/internal/api/rest"
	"nexora/internal/pkg/logger"
)

type Handler struct {
	svc *Service
	log *logger.Logger
}

func NewHandler(svc *Service, log *logger.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Register(ctx *rest.Context) {
	var req RegisterRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	resp, err := h.svc.Register(ctx.Request.Context(), req)
	if err != nil {
		switch err {
		case ErrEmailAlreadyExists:
			ctx.Error(http.StatusConflict, "EMAIL_EXISTS", err.Error())
		case ErrInvalidCredentials:
			ctx.Error(http.StatusBadRequest, "INVALID_INPUT", err.Error())
		default:
			h.log.Error("registration failed", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "registration failed")
		}
		return
	}

	ctx.JSON(http.StatusCreated, resp)
}

func (h *Handler) Login(ctx *rest.Context) {
	var req LoginRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "email and password are required")
		return
	}

	resp, err := h.svc.Login(ctx.Request.Context(), req)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			ctx.Error(http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error())
		case ErrMFARequired:
			ctx.JSON(http.StatusOK, map[string]string{"status": "mfa_required", "message": "MFA code required"})
		case ErrInvalidMFACode:
			ctx.Error(http.StatusUnauthorized, "INVALID_MFA", err.Error())
		default:
			h.log.Error("login failed", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "login failed")
		}
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) RefreshToken(ctx *rest.Context) {
	var req RefreshRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "refresh token is required")
		return
	}

	resp, err := h.svc.RefreshToken(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		switch err {
		case ErrInvalidToken, ErrSessionExpired:
			ctx.Error(http.StatusUnauthorized, "INVALID_TOKEN", err.Error())
		case ErrUserNotFound:
			ctx.Error(http.StatusUnauthorized, "USER_NOT_FOUND", err.Error())
		default:
			h.log.Error("token refresh failed", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "token refresh failed")
		}
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) Logout(ctx *rest.Context) {
	userID, ok := GetUserIDFromCtx(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	bodyRefreshToken := ""
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := ctx.Decode(&body); err == nil {
		bodyRefreshToken = body.RefreshToken
	}

	if err := h.svc.Logout(ctx.Request.Context(), userID, bodyRefreshToken); err != nil {
		h.log.Error("logout failed", "error", err)
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "logged_out"})
}

func (h *Handler) GetOAuthURL(ctx *rest.Context) {
	provider := ctx.Request.URL.Query().Get("provider")
	if provider == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "provider is required")
		return
	}

	redirect := ctx.Request.URL.Query().Get("redirect_uri")

	url, err := h.svc.oauthService.GetAuthorizationURL(provider, redirect)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_PROVIDER", err.Error())
		return
	}

	ctx.JSON(http.StatusOK, OAuthURLResponse{URL: url})
}

func (h *Handler) OAuthCallback(ctx *rest.Context) {
	var req OAuthRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Provider == "" || req.Code == "" {
		ctx.Error(http.StatusBadRequest, "INVALID_INPUT", "provider and code are required")
		return
	}

	resp, err := h.svc.HandleOAuthCallback(ctx.Request.Context(), req.Provider, req.Code)
	if err != nil {
		switch err {
		case ErrOAuthProviderError:
			ctx.Error(http.StatusBadGateway, "OAUTH_ERROR", err.Error())
		case ErrOAuthEmailExists:
			ctx.Error(http.StatusConflict, "EMAIL_EXISTS", err.Error())
		default:
			h.log.Error("oauth callback failed", "error", err)
			ctx.Error(http.StatusInternalServerError, "INTERNAL", "oauth authentication failed")
		}
		return
	}
	if resp == nil {
		ctx.Error(http.StatusInternalServerError, "OAUTH_ERROR", "user could not be created")
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) EnrollMFA(ctx *rest.Context) {
	userID, ok := GetUserIDFromCtx(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	enroll, err := h.svc.GetMFAService().Enroll(ctx.Request.Context(), userID, h.svc)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "MFA_ERROR", err.Error())
		return
	}

	ctx.JSON(http.StatusOK, enroll)
}

func (h *Handler) VerifyMFA(ctx *rest.Context) {
	userID, ok := GetUserIDFromCtx(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req MFAVerifyRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if err := h.svc.GetMFAService().VerifyAndEnable(ctx.Request.Context(), userID, req.Code, h.svc); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_CODE", "invalid MFA code")
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "mfa_enabled"})
}

func (h *Handler) Me(ctx *rest.Context) {
	userID, ok := GetUserIDFromCtx(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	user, err := h.svc.GetUserByID(ctx.Request.Context(), userID)
	if err != nil {
		ctx.Error(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		return
	}

	ctx.JSON(http.StatusOK, user)
}

func (h *Handler) DisableMFA(ctx *rest.Context) {
	userID, ok := GetUserIDFromCtx(ctx.Request.Context())
	if !ok {
		ctx.Error(http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req MFADisableRequest
	if err := ctx.Decode(&req); err != nil {
		ctx.Error(http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if err := h.svc.GetMFAService().Disable(ctx.Request.Context(), userID, req.Password, h.svc); err != nil {
		ctx.Error(http.StatusBadRequest, "MFA_DISABLE_FAILED", err.Error())
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "mfa_disabled"})
}
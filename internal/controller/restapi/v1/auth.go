package v1

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/validation"
	"mini-instagram/pkg/image"
)

func (h *V1) login(c *gin.Context) {
	var req request.Login
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Info("login request parsing failed", "error", err)
		h.handleError(c, apihttp.BadRequest, "invalid JSON request")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if err := validation.Login(req); err != nil {
		h.handleUsecaseError(c, err, "login validation failed", "email", req.Email)
		return
	}

	accessToken, err := h.auth.Login(c.Request.Context(), req)
	if err != nil {
		h.handleUsecaseError(c, err, "login failed", "email", req.Email)
		return
	}

	h.handleResponse(c, apihttp.OK, gin.H{"access_token": accessToken})
}

func (h *V1) signUp(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(image.DefaultMaxSize); err != nil {
		h.logger.Info("signup request parsing failed", "error", err)
		h.handleError(c, apihttp.BadRequest, "invalid multipart request")
		return
	}

	req := request.SignUp{
		Email:    strings.TrimSpace(strings.ToLower(c.Request.FormValue("email"))),
		FullName: strings.TrimSpace(c.Request.FormValue("full_name")),
		Username: strings.TrimSpace(strings.ToLower(c.Request.FormValue("username"))),
		Bio:      strings.TrimSpace(c.Request.FormValue("bio")),
		Password: c.Request.FormValue("password"),
	}
	if err := validation.SignUp(req); err != nil {
		h.handleUsecaseError(c, err, "signup validation failed", "email", req.Email)
		return
	}

	if err := h.auth.CheckSignUpAvailability(c.Request.Context(), req.Email, req.Username); err != nil {
		h.handleUsecaseError(c, err, "signup availability check failed", "email", req.Email)
		return
	}

	avatarPath, err := h.saveAvatar(c)
	if err != nil {
		h.logger.Error("signup avatar upload failed", "email", req.Email, "error", err)
		h.handleError(c, apihttp.BadRequest, err.Error())
		return
	}

	accessToken, err := h.auth.SignUp(c.Request.Context(), request.SignUp{
		Email:      req.Email,
		FullName:   req.FullName,
		Username:   req.Username,
		Bio:        req.Bio,
		Password:   req.Password,
		AvatarPath: avatarPath,
	})
	if err != nil {
		h.handleUsecaseError(c, err, "signup failed", "email", req.Email)
		return
	}

	h.handleResponse(c, apihttp.OK, gin.H{"access_token": accessToken})
}

func (h *V1) logout(c *gin.Context) {
	h.handleResponse(c, apihttp.OK, nil)
}

func (h *V1) saveAvatar(c *gin.Context) (string, error) {
	file, header, err := c.Request.FormFile("avatar")
	if errors.Is(err, http.ErrMissingFile) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer file.Close()

	return image.Save(file, header, h.storage, "avatars", image.DefaultMaxSize)
}

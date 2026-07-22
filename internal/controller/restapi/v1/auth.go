package v1

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/entity"
	"mini-instagram/pkg/image"
)

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
	if field, message := validateSignUp(req); field != "" {
		h.logger.Info("signup validation failed", "field", field)
		h.handleError(c, apihttp.BadRequest, message)
		return
	}

	if err := h.auth.CheckSignUpAvailability(c.Request.Context(), req.Email, req.Username); err != nil {
		h.writeSignUpError(c, err)
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
		h.writeSignUpError(c, err)
		return
	}

	h.handleResponse(c, apihttp.OK, gin.H{"access_token": accessToken})
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

func (h *V1) writeSignUpError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrEmailTaken):
		h.logger.Info("signup rejected", "field", "email", "reason", "already exists")
		h.handleError(c, apihttp.Conflict, "email already exists")
	case errors.Is(err, entity.ErrUsernameTaken):
		h.logger.Info("signup rejected", "field", "username", "reason", "already exists")
		h.handleError(c, apihttp.Conflict, "username already exists")
	default:
		h.logger.Error("signup failed", "error", err)
		h.handleError(c, apihttp.InternalServerError, "could not sign up")
	}
}

func validateSignUp(req request.SignUp) (string, string) {
	switch {
	case req.Email == "":
		return "email", "email is required"
	case req.FullName == "":
		return "full_name", "full_name is required"
	case req.Username == "":
		return "username", "username is required"
	case req.Password == "":
		return "password", "password is required"
	case len(req.Password) < 8:
		return "password", "password must be at least 8 characters"
	}
	return "", ""
}

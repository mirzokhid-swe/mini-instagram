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

// login godoc
//
//	@Summary		Log in
//	@Description	Authenticates a user by email/password and returns a JWT access token. Rate-limited to 5 requests/minute per email.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.Login	true	"Login credentials"
//	@Success		200		{object}	http.Response{data=object{access_token=string}}
//	@Failure		400		{object}	http.Response	"validation error"
//	@Failure		401		{object}	http.Response	"invalid email or password"
//	@Failure		429		{object}	http.Response	"too many attempts"
//	@Router			/auth/login [post]
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

// signUp godoc
//
//	@Summary		Sign up
//	@Description	Creates a new user account and returns a JWT access token. Rate-limited to 5 requests/minute per email.
//	@Tags			auth
//	@Accept			mpfd
//	@Produce		json
//	@Param			email		formData	string	true	"Email address"
//	@Param			full_name	formData	string	true	"Full name"
//	@Param			username	formData	string	true	"Username (lowercase, 3-32 chars, [a-z0-9_.])"
//	@Param			password	formData	string	true	"Password (min 8 chars)"
//	@Param			bio			formData	string	false	"Short bio"
//	@Param			avatar		formData	file	false	"Avatar image (jpeg/png/webp)"
//	@Success		200			{object}	http.Response{data=object{access_token=string}}
//	@Failure		400			{object}	http.Response	"validation error"
//	@Failure		409			{object}	http.Response	"email or username already taken"
//	@Failure		429			{object}	http.Response	"too many attempts"
//	@Router			/auth/sign-up [post]
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

// logout godoc
//
//	@Summary		Log out
//	@Description	Verifies the access token; the client is expected to discard it afterwards (stateless JWT, no server-side blacklist).
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	http.Response
//	@Failure		401	{object}	http.Response	"missing or invalid token"
//	@Router			/auth/logout [post]
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

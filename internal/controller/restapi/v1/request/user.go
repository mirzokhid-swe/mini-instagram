package request

import "mime/multipart"

// UpdateProfile holds the inputs for PUT /profile.
type UpdateProfile struct {
	UserID       int64
	Username     string
	FullName     string
	Bio          string
	Avatar       multipart.File
	AvatarHeader *multipart.FileHeader
}

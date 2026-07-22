package request

import "mime/multipart"

// CreatePost holds the inputs required by the post creation use case.
type CreatePost struct {
	UserID  int64
	Caption string
	File    multipart.File
	Header  *multipart.FileHeader
}

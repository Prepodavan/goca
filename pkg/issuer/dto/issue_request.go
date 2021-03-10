package dto

import "mime/multipart"

type IssueRequest struct {
	Days   uint64                `form:"days" binding:"required,min=1"`
	CSR    *multipart.FileHeader `form:"csr" binding:"required_without=Config"`
	Config *multipart.FileHeader `form:"config" binding:"required_without=CSR"`
}

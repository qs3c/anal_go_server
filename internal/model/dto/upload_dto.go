package dto

// ParseUploadResponse 解析上传文件的响应
type ParseUploadResponse struct {
	UploadID     string       `json:"upload_id"`
	ExpiresAt    string       `json:"expires_at"`
	Files        []GoFileInfo `json:"files"`
	TotalFiles   int          `json:"total_files"`
	TotalStructs int          `json:"total_structs"`
}

// GoFileInfo Go 文件信息
type GoFileInfo struct {
	Path    string   `json:"path"`
	Structs []string `json:"structs"`
}

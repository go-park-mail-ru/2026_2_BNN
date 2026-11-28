package onlineconf

const (
	ScopeFrontend = "frontend"
	ScopeBackend  = "backend"
	ScopeAuth     = "auth"
	ScopeNotes    = "notes"
)

const (
	KindConfig = "config"
	KindFlag   = "flag"
)

const (
	ValueString  = "string"
	ValueNumber  = "number"
	ValueBoolean = "boolean"
	ValueObject  = "object"
	ValueArray   = "array"
)

const (
	KeyAttachmentsMaxFileSize   = "attachments.max_file_size_bytes"
	KeyAttachmentsUploadEnabled = "attachments.upload_enabled"
)

const (
	MinAttachmentSizeBytes = 1 << 20
	MaxAttachmentSizeBytes = 10 << 20
)

const (
	CodeForbidden       = "FORBIDDEN"
	CodeValidation      = "VALIDATION"
	CodeFeatureDisabled = "FEATURE_DISABLED"
	CodeFileTooLarge    = "FILE_TOO_LARGE"
)

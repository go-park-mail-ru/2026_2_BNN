package onlineconf

const (
	CodeRolloutPlanExists     = "ROLLOUT_PLAN_EXISTS"
	CodeRolloutPlanInUse      = "ROLLOUT_PLAN_IN_USE"
	CodeRolloutNotReady       = "ROLLOUT_NOT_READY"
	CodeScheduleFinished      = "SCHEDULE_FINISHED"
	CodeRolloutStepNotAllowed = "ROLLOUT_STEP_NOT_ALLOWED"
)

const (
	RolloutWaiting         = "waiting"
	RolloutPendingApproval = "pending_approval"
	RolloutDone            = "done"
	RolloutUnassigned      = "unassigned"
)

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

const (
	RolloutPercentMin = 0
	RolloutPercentMax = 100

	RolloutPlanMinSteps      = 2
	RolloutPlanNameMaxLength = 100
)

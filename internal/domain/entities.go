package domain

import "time"

type Project struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Folder struct {
	ID        int64     `json:"id"`
	ProjectID int64     `json:"project_id"`
	ParentID  *int64    `json:"parent_id"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type File struct {
	ID        int64     `json:"id"`
	ProjectID int64     `json:"project_id"`
	FolderID  *int64    `json:"folder_id"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FileVersion struct {
	ID        int64     `json:"id"`
	FileID    int64     `json:"file_id"`
	Content   string    `json:"content"`
	AstHash   string    `json:"ast_hash"`
	IsLatest  bool      `json:"is_latest"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
}

type Feature struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Entity struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CodeStyle struct {
	ID        int64     `json:"id"`
	ProjectID int64     `json:"project_id"`
	Rule      string    `json:"rule"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StyleAnomaly struct {
	ID            int64     `json:"id"`
	FileVersionID int64     `json:"file_version_id"`
	CodeStyleID   int64     `json:"code_style_id"`
	Rationale     string    `json:"rationale"`
	CreatedAt     time.Time `json:"created_at"`
}

type Job struct {
	ID         int64     `json:"id"`
	ProjectID  int64     `json:"project_id"`
	TargetID   int64     `json:"target_id"` // e.g. FileVersionID or EntityID depending on Stage
	TargetType string    `json:"target_type"`
	Stage      Stage     `json:"stage"`
	Status     JobStatus `json:"status"`
	Priority   int       `json:"priority"`
	RetryCount int       `json:"retry_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type LLMConfig struct {
	ID             int64     `json:"id"`
	ProjectID      int64     `json:"project_id"`
	ModelTier      ModelTier `json:"model_tier"`
	SafeTokenLimit int       `json:"safe_token_limit"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type FeatureInteraction struct {
	ID            int64     `json:"id"`
	FromFeatureID int64     `json:"from_feature_id"`
	ToFeatureID   int64     `json:"to_feature_id"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}

type MacroPipeline struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ApiEndpoint struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	Path        string    `json:"path"`
	Method      string    `json:"method"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PromptTemplate struct {
	ID        int64     `json:"id"`
	Stage     Stage     `json:"stage"`
	Template  string    `json:"template"`
	Version   int       `json:"version"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DeadLetterItem struct {
	ID        int64     `json:"id"`
	JobID     int64     `json:"job_id"`
	Payload   string    `json:"payload"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// Junction Structs

type FileFeature struct {
	FileVersionID int64 `json:"file_version_id"`
	FeatureID     int64 `json:"feature_id"`
}

type FileStyle struct {
	FileVersionID int64 `json:"file_version_id"`
	CodeStyleID   int64 `json:"code_style_id"`
}

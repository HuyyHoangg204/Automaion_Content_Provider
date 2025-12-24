package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Script represents a script/workflow for a topic (1-1 với user + topic)
type Script struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TopicID   string    `json:"topic_id" gorm:"not null;index;type:uuid"`
	UserID    string    `json:"user_id" gorm:"not null;index;type:uuid"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Topic    Topic           `json:"topic,omitempty" gorm:"foreignKey:TopicID;references:ID;constraint:OnDelete:CASCADE"`
	User     User            `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Projects []ScriptProject `json:"projects,omitempty" gorm:"foreignKey:ScriptID;references:ID;constraint:OnDelete:CASCADE;order:created_at_db"`
	Edges    []ScriptEdge    `json:"edges,omitempty" gorm:"foreignKey:ScriptID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Script) TableName() string {
	return "scripts"
}

// StringArray is a custom type for storing []string in JSONB column
type StringArray []string

// Value implements driver.Valuer interface for GORM
func (sa StringArray) Value() (driver.Value, error) {
	if len(sa) == 0 {
		return "[]", nil
	}
	return json.Marshal(sa)
}

// Scan implements sql.Scanner interface for GORM
func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = StringArray{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	if len(bytes) == 0 {
		*sa = StringArray{}
		return nil
	}

	return json.Unmarshal(bytes, sa)
}

// ScriptProject represents a project/node in a script
// Composite primary key: (script_id, project_id) - project_id chỉ cần unique trong scope của một script
type ScriptProject struct {
	ScriptID  string `json:"script_id" gorm:"primaryKey;type:uuid;not null"`          // Part of composite primary key
	ProjectID string `json:"project_id" gorm:"primaryKey;type:varchar(255);not null"` // Frontend ID (timestamp) - Part of composite primary key
	Name      string `json:"name" gorm:"type:varchar(255);not null"`
	Filename  string `json:"filename,omitempty" gorm:"type:varchar(255)"` // Tên file (kết quả/merge) gắn với project
	// Note: gem_name không lưu trong DB, được generate từ name với format {username}_{name} khi cần
	Description     string    `json:"description,omitempty" gorm:"type:text"`             // Mô tả cho gem/project
	Instructions    string    `json:"instructions,omitempty" gorm:"type:text"`            // Instruction cho gem/project
	GeminiAccountID *string   `json:"gemini_account_id,omitempty" gorm:"type:uuid;index"` // Gemini account cho gem này
	CreatedAt       time.Time `json:"created_at" gorm:"not null"`                         // Từ frontend
	CreatedAtDB     time.Time `json:"created_at_db" gorm:"default:now()"`                 // Timestamp khi lưu vào DB
	UpdatedAt       time.Time `json:"updated_at"`

	// Relationships
	Script  Script         `json:"script,omitempty" gorm:"foreignKey:ScriptID;references:ID;constraint:OnDelete:CASCADE"`
	Prompts []ScriptPrompt `json:"prompts,omitempty" gorm:"foreignKey:ProjectID;references:ProjectID;constraint:OnDelete:CASCADE;order:prompt_order"`
}

func (ScriptProject) TableName() string {
	return "script_projects"
}

// ScriptPrompt represents a prompt in a project
type ScriptPrompt struct {
	ID          string      `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ScriptID    string      `json:"script_id" gorm:"not null;index;type:uuid"`          // Cần để reference đến ScriptProject composite key
	ProjectID   string      `json:"project_id" gorm:"not null;index;type:varchar(255)"` // Frontend project_id (timestamp), part of ScriptProject composite key
	PromptText  string      `json:"text" gorm:"type:text;not null"`
	Filename    string      `json:"filename,omitempty" gorm:"type:varchar(255)"` // Tên file output gắn với prompt
	InputFiles  StringArray `json:"input_files,omitempty" gorm:"type:jsonb"`     // Danh sách file name dùng làm input cho prompt này
	Exit        bool        `json:"exit" gorm:"default:false"`
	Merge       bool        `json:"merge" gorm:"default:false"` // New field: Merge results
	PromptOrder int         `json:"prompt_order" gorm:"not null;default:0"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`

	// Relationships - Note: No foreign key constraint because composite key reference is complex
	// We rely on application logic to maintain referential integrity
	// (ScriptID, ProjectID) references ScriptProject(ScriptID, ProjectID)
	Project ScriptProject `json:"project,omitempty" gorm:"foreignKey:ScriptID,ProjectID;references:ScriptID,ProjectID"`
}

func (ScriptPrompt) TableName() string {
	return "script_prompts"
}

// ScriptEdge represents a connection between projects
type ScriptEdge struct {
	ID              string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ScriptID        string    `json:"script_id" gorm:"not null;index;type:uuid"`
	EdgeID          string    `json:"edge_id" gorm:"type:varchar(255);not null"` // Frontend ID (format "edge-{source}-{target}")
	SourceProjectID string    `json:"source" gorm:"type:varchar(255);not null"`  // project_id
	TargetProjectID string    `json:"target" gorm:"type:varchar(255);not null"`  // project_id
	SourceName      string    `json:"sourceName,omitempty" gorm:"type:varchar(255)"`
	TargetName      string    `json:"targetName,omitempty" gorm:"type:varchar(255)"`
	CreatedAt       time.Time `json:"created_at"`

	// Relationships
	Script Script `json:"script,omitempty" gorm:"foreignKey:ScriptID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ScriptEdge) TableName() string {
	return "script_edges"
}

// SaveScriptRequest represents the request to save a script
type SaveScriptRequest struct {
	Projects []ScriptProjectRequest `json:"projects" binding:"required,min=1"`
	Edges    []ScriptEdgeRequest    `json:"edges" binding:"required"`
}

type ScriptProjectRequest struct {
	ID         string `json:"id" binding:"required"` // Frontend ID (timestamp)
	Name       string `json:"name" binding:"required"`
	OutputName string `json:"output_name,omitempty"` // Tên file merge/output của project (phase1_merged, v.v.)
	// Note: gem_name không cần truyền, được generate từ name với format {username}_{name} khi cần
	Description     string                `json:"description,omitempty"`         // Mô tả cho gem/project
	Instructions    string                `json:"instructions,omitempty"`        // Instruction cho gem/project
	GeminiAccountID string                `json:"gemini_account_id,omitempty"`   // Gemini account ID cho gem này
	CreatedAt       string                `json:"created_at" binding:"required"` // ISO 8601 timestamp
	Prompts         []ScriptPromptRequest `json:"prompts" binding:"required,min=1"`
}

type ScriptPromptRequest struct {
	Text       string   `json:"text" binding:"required"`
	Filename   string   `json:"filename,omitempty"`
	InputFiles []string `json:"input_files,omitempty"`
	Exit       bool     `json:"exit"`
	Merge      bool     `json:"merge"`
}

type ScriptEdgeRequest struct {
	ID         string `json:"id" binding:"required"`     // Frontend ID (format "edge-{source}-{target}")
	Source     string `json:"source" binding:"required"` // project_id
	Target     string `json:"target" binding:"required"` // project_id
	SourceName string `json:"sourceName,omitempty"`
	TargetName string `json:"targetName,omitempty"`
}

// ScriptResponse represents the response for script operations
type ScriptResponse struct {
	ID        string                  `json:"id"`
	TopicID   string                  `json:"topic_id"`
	UserID    string                  `json:"user_id"`
	Projects  []ScriptProjectResponse `json:"projects"`
	Edges     []ScriptEdgeResponse    `json:"edges"`
	CreatedAt string                  `json:"created_at"`
	UpdatedAt string                  `json:"updated_at"`
}

type ScriptProjectResponse struct {
	ProjectID string `json:"project_id"` // Frontend ID (timestamp) - chỉ cần field này
	Name      string `json:"name"`
	Filename  string `json:"filename,omitempty"`
	// Note: gem_name không trả về trong response, được generate từ name khi cần
	Description     string                 `json:"description,omitempty"`
	Instructions    string                 `json:"instructions,omitempty"`
	GeminiAccountID *string                `json:"gemini_account_id,omitempty"`
	CreatedAt       string                 `json:"created_at"` // Từ frontend
	Prompts         []ScriptPromptResponse `json:"prompts"`
}

type ScriptPromptResponse struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Filename    string   `json:"filename,omitempty"`
	InputFiles  []string `json:"input_files,omitempty"`
	Exit        bool     `json:"exit"`
	Merge       bool     `json:"merge"`
	PromptOrder int      `json:"prompt_order"`
}

type ScriptEdgeResponse struct {
	ID         string `json:"id"`      // UUID từ DB
	EdgeID     string `json:"edge_id"` // Frontend ID
	Source     string `json:"source"`
	Target     string `json:"target"`
	SourceName string `json:"sourceName,omitempty"`
	TargetName string `json:"targetName,omitempty"`
}

// ScriptExecution represents an execution instance of a script
type ScriptExecution struct {
	ID               string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ScriptID         string     `json:"script_id" gorm:"not null;index;type:uuid"`
	TopicID          string     `json:"topic_id" gorm:"not null;index;type:uuid"`
	UserID           string     `json:"user_id" gorm:"not null;index;type:uuid"`
	Status           string     `json:"status" gorm:"type:varchar(20);not null;default:'pending';index"` // pending, running, completed, failed, cancelled
	CurrentProjectID *string    `json:"current_project_id,omitempty" gorm:"type:varchar(255)"`
	TunnelURL        string     `json:"tunnel_url,omitempty" gorm:"type:varchar(500)"` // TunnelURL từ launch response
	DebugPort        int        `json:"debug_port,omitempty" gorm:"default:0"`         // DebugPort từ Chrome launch response
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	ErrorMessage     string     `json:"error_message,omitempty" gorm:"type:text"`
	RetryCount       int        `json:"retry_count" gorm:"default:0"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Relationships
	Script Script `json:"script,omitempty" gorm:"foreignKey:ScriptID;references:ID;constraint:OnDelete:CASCADE"`
	Topic  Topic  `json:"topic,omitempty" gorm:"foreignKey:TopicID;references:ID;constraint:OnDelete:CASCADE"`
	User   User   `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ScriptExecution) TableName() string {
	return "script_executions"
}

// ExecuteScriptRequest represents the request to execute a script
type ExecuteScriptRequest struct {
	// No fields needed - script is identified by topic_id and user_id
}

// ExecuteScriptResponse represents the response for script execution
type ExecuteScriptResponse struct {
	ExecutionID string `json:"execution_id"`
	ScriptID    string `json:"script_id"`
	TopicID     string `json:"topic_id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

// ScriptProjectExecution tracks execution status of each project in a script execution
type ScriptProjectExecution struct {
	ID           string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ExecutionID  string     `json:"execution_id" gorm:"not null;index;type:uuid"`
	ProjectID    string     `json:"project_id" gorm:"not null;index;type:varchar(255)"`
	ProjectOrder int        `json:"project_order" gorm:"not null"`                                   // Thứ tự trong execution (0-based)
	Status       string     `json:"status" gorm:"type:varchar(20);not null;default:'pending';index"` // pending, running, completed, failed
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty" gorm:"type:text"`
	RetryCount   int        `json:"retry_count" gorm:"default:0"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relationships
	Execution ScriptExecution `json:"execution,omitempty" gorm:"foreignKey:ExecutionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ScriptProjectExecution) TableName() string {
	return "script_project_executions"
}

// CreateProjectRequest represents the request to create a single project (and its gem)
type CreateProjectRequest struct {
	Name         string `json:"name" binding:"required"` // Project name (gem_name sẽ tự generate từ name với prefix username)
	Description  string `json:"description,omitempty"`   // Mô tả cho gem/project
	Instructions string `json:"instructions,omitempty"`  // Instruction cho gem/project
	// Note: gem_name được tự động generate từ name với format: {username}_{name} (giống logic cũ)
	// Note: knowledge_files được tự động lấy từ cache (files vừa upload), không cần truyền từ API
	// Note: gemini_account_id được tự động chọn bởi system, không cần truyền từ API
}

// CreateProjectResponse represents the response for project creation
type CreateProjectResponse struct {
	ProjectID string `json:"project_id"` // Frontend ID (timestamp)
	Name      string `json:"name"`
	// Note: gem_name không trả về, được generate từ name với format {username}_{name} khi cần
	Description  string `json:"description,omitempty"`
	Instructions string `json:"instructions,omitempty"`
	CreatedAt    string `json:"created_at"`
	// Note: gemini_account_id không trả về trong response, được quản lý nội bộ
}

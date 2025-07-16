package errors

import "fmt"

// SyncError represents a sync-related error
type SyncError struct {
	Type    string
	Message string
	Err     error
}

func (e *SyncError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *SyncError) Unwrap() error {
	return e.Err
}

// Error type constants
const (
	ErrTypeValidation = "validation"
	ErrTypeNetwork    = "network"
	ErrTypeAuth       = "authentication"
	ErrTypeFileSystem = "filesystem"
	ErrTypeTimeout    = "timeout"
	ErrTypeUnknown    = "unknown"
)

// NewValidationError creates a new validation error
func NewValidationError(message string) *SyncError {
	return &SyncError{
		Type:    ErrTypeValidation,
		Message: message,
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(message string, err error) *SyncError {
	return &SyncError{
		Type:    ErrTypeNetwork,
		Message: message,
		Err:     err,
	}
}

// NewAuthError creates a new authentication error
func NewAuthError(message string, err error) *SyncError {
	return &SyncError{
		Type:    ErrTypeAuth,
		Message: message,
		Err:     err,
	}
}

// NewFileSystemError creates a new filesystem error
func NewFileSystemError(message string, err error) *SyncError {
	return &SyncError{
		Type:    ErrTypeFileSystem,
		Message: message,
		Err:     err,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message string, err error) *SyncError {
	return &SyncError{
		Type:    ErrTypeTimeout,
		Message: message,
		Err:     err,
	}
}

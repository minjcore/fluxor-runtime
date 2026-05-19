package file

import "fmt"

// ErrSecretNotFound is returned when a secret key is not found
type ErrSecretNotFound struct {
	Key string
}

func (e *ErrSecretNotFound) Error() string {
	return fmt.Sprintf("secret not found: %s", e.Key)
}

// IsSecretNotFound checks if an error is ErrSecretNotFound
func IsSecretNotFound(err error) bool {
	_, ok := err.(*ErrSecretNotFound)
	return ok
}

// ErrInvalidFileFormat is returned when the file format is invalid
type ErrInvalidFileFormat struct {
	Path   string
	Format string
	Err    error
}

func (e *ErrInvalidFileFormat) Error() string {
	return fmt.Sprintf("invalid file format %s for file %s: %v", e.Format, e.Path, e.Err)
}

func (e *ErrInvalidFileFormat) Unwrap() error {
	return e.Err
}

// IsInvalidFileFormat checks if an error is ErrInvalidFileFormat
func IsInvalidFileFormat(err error) bool {
	_, ok := err.(*ErrInvalidFileFormat)
	return ok
}

// ErrFileNotReadable is returned when the file cannot be read
type ErrFileNotReadable struct {
	Path string
	Err  error
}

func (e *ErrFileNotReadable) Error() string {
	return fmt.Sprintf("file not readable: %s: %v", e.Path, e.Err)
}

func (e *ErrFileNotReadable) Unwrap() error {
	return e.Err
}

// IsFileNotReadable checks if an error is ErrFileNotReadable
func IsFileNotReadable(err error) bool {
	_, ok := err.(*ErrFileNotReadable)
	return ok
}

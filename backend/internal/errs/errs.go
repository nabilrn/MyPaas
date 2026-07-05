package errs

import "errors"

var (
	ErrBadRequest          = errors.New("bad request")
	ErrComposeFileNotFound = errors.New("compose file not found")
	ErrComposeUnsupported  = errors.New("compose action is not supported")
	ErrDockerfileNotFound  = errors.New("dockerfile not found")
	ErrEmailNotWhitelisted = errors.New("email not in whitelist")
	ErrForbidden           = errors.New("forbidden")
	ErrNoDeployConfig      = errors.New("no deploy config found")
	ErrNotFound            = errors.New("not found")
	ErrPortPoolExhausted   = errors.New("port pool exhausted")
	ErrProjectNameTaken    = errors.New("project name already taken")
	ErrQuotaExceeded       = errors.New("quota exceeded")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrValidation          = errors.New("validation failed")
)

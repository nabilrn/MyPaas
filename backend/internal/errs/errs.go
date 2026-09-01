package errs

import "errors"

var (
	ErrBadRequest                  = errors.New("bad request")
	ErrCannotRemoveMaster          = errors.New("master user cannot be removed")
	ErrCannotRemoveOwner           = errors.New("owner users cannot remove one another")
	ErrComposeFileNotFound         = errors.New("compose file not found")
	ErrComposeUnsupported          = errors.New("compose action is not supported")
	ErrContainerNotFound           = errors.New("container not found")
	ErrContainerRunning            = errors.New("running container cannot be removed")
	ErrDockerfileNotFound          = errors.New("dockerfile not found")
	ErrEmailNotWhitelisted         = errors.New("email not in whitelist")
	ErrForbidden                   = errors.New("forbidden")
	ErrGitHubAuthorizationRequired = errors.New("github authorization required")
	ErrNoDeployConfig              = errors.New("no deploy config found")
	ErrNotFound                    = errors.New("not found")
	ErrPortPoolExhausted           = errors.New("port pool exhausted")
	ErrProjectNameTaken            = errors.New("project name already taken")
	ErrQuotaExceeded               = errors.New("quota exceeded")
	ErrShellInputTooLarge          = errors.New("shell input is too large")
	ErrShellSessionClosed          = errors.New("shell session is closed")
	ErrShellSessionNotFound        = errors.New("shell session not found")
	ErrUnauthorized                = errors.New("unauthorized")
	ErrUserAlreadyExists           = errors.New("user already exists")
	ErrValidation                  = errors.New("validation failed")
)

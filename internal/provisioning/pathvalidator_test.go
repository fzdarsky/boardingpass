package provisioning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPathValidator(t *testing.T) {
	allowedPaths := []string{"/etc/systemd/", "/etc/myapp/"}
	validator := NewPathValidator(allowedPaths)

	assert.NotNil(t, validator)
	assert.Equal(t, allowedPaths, validator.allowedPaths)
}

func TestPathValidator_ValidatePath(t *testing.T) {
	validator := NewPathValidator([]string{
		"/etc/systemd/",
		"/etc/NetworkManager/",
		"/etc/myapp/",
	})

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "path cannot be empty",
		},
		{
			name:    "path with traversal (..)",
			path:    "systemd/../passwd",
			wantErr: true,
			errMsg:  "contains path traversal sequence",
		},
		{
			name:    "path with multiple traversals",
			path:    "../../etc/shadow",
			wantErr: true,
			errMsg:  "contains path traversal sequence",
		},
		{
			name:    "valid path in systemd directory",
			path:    "systemd/system/myservice.service",
			wantErr: false,
		},
		{
			name:    "valid path in NetworkManager directory",
			path:    "NetworkManager/system-connections/eth0.nmconnection",
			wantErr: false,
		},
		{
			name:    "valid path in myapp directory",
			path:    "myapp/config.yaml",
			wantErr: false,
		},
		{
			name:    "path not in allow-list (passwd)",
			path:    "passwd",
			wantErr: true,
			errMsg:  "is not in allow-list",
		},
		{
			name:    "path not in allow-list (shadow)",
			path:    "shadow",
			wantErr: true,
			errMsg:  "is not in allow-list",
		},
		{
			name:    "path not in allow-list (sudoers)",
			path:    "sudoers",
			wantErr: true,
			errMsg:  "is not in allow-list",
		},
		{
			name:    "path in disallowed subdirectory",
			path:    "cron.d/malicious",
			wantErr: true,
			errMsg:  "is not in allow-list",
		},
		{
			name:    "nested path in allowed directory",
			path:    "systemd/system/multi-user.target.wants/myservice.service",
			wantErr: false,
		},
		{
			name:    "path with leading slash (should still work)",
			path:    "/systemd/system/myservice.service",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPathValidator_ValidateAll(t *testing.T) {
	validator := NewPathValidator([]string{"/etc/systemd/", "/etc/myapp/"})

	tests := []struct {
		name    string
		paths   []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "all valid paths",
			paths:   []string{"systemd/system/test.service", "myapp/config.yaml"},
			wantErr: false,
		},
		{
			name:    "one invalid path",
			paths:   []string{"systemd/system/test.service", "passwd"},
			wantErr: true,
			errMsg:  "is not in allow-list",
		},
		{
			name:    "empty list",
			paths:   []string{},
			wantErr: false,
		},
		{
			name:    "first path invalid",
			paths:   []string{"shadow", "systemd/system/test.service"},
			wantErr: true,
			errMsg:  "is not in allow-list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateAll(tt.paths)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPathValidator_EdgeCases(t *testing.T) {
	validator := NewPathValidator([]string{"/etc/systemd/"})

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "path exactly matching allowed directory (no trailing content)",
			path:    "systemd",
			wantErr: false,
		},
		{
			name:    "path with redundant slashes",
			path:    "systemd//system//test.service",
			wantErr: false,
		},
		{
			name:    "path with current directory reference",
			path:    "systemd/./system/test.service",
			wantErr: false,
		},
		{
			name:    "attempt to escape with absolute path",
			path:    "/etc/systemd/../../etc/passwd",
			wantErr: true,
			errMsg:  "contains path traversal sequence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPathValidator_MultipleAllowedPaths(t *testing.T) {
	validator := NewPathValidator([]string{
		"/etc/systemd/system/",
		"/etc/systemd/network/",
		"/etc/NetworkManager/system-connections/",
	})

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "allowed in systemd/system",
			path:    "systemd/system/test.service",
			wantErr: false,
		},
		{
			name:    "allowed in systemd/network",
			path:    "systemd/network/10-eth0.network",
			wantErr: false,
		},
		{
			name:    "allowed in NetworkManager",
			path:    "NetworkManager/system-connections/eth0.nmconnection",
			wantErr: false,
		},
		{
			name:    "not allowed in systemd/user (not in list)",
			path:    "systemd/user/test.service",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

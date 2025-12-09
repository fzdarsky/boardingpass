package provisioning

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBundle(t *testing.T) {
	tests := []struct {
		name    string
		bundle  *protocol.ConfigBundle
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil bundle",
			bundle:  nil,
			wantErr: true,
			errMsg:  "bundle cannot be nil",
		},
		{
			name: "empty files list",
			bundle: &protocol.ConfigBundle{
				Files: []protocol.ConfigFile{},
			},
			wantErr: true,
			errMsg:  "bundle must contain at least one file",
		},
		{
			name: "valid bundle with one file",
			bundle: &protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    "test.conf",
						Content: base64.StdEncoding.EncodeToString([]byte("test content")),
						Mode:    0o644,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "file with empty path",
			bundle: &protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    "",
						Content: base64.StdEncoding.EncodeToString([]byte("content")),
						Mode:    0o644,
					},
				},
			},
			wantErr: true,
			errMsg:  "file at index 0 has empty path",
		},
		{
			name: "file with empty content",
			bundle: &protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    "test.conf",
						Content: "",
						Mode:    0o644,
					},
				},
			},
			wantErr: true,
			errMsg:  "file test.conf has empty content",
		},
		{
			name: "file with invalid Base64",
			bundle: &protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    "test.conf",
						Content: "not-valid-base64!!!",
						Mode:    0o644,
					},
				},
			},
			wantErr: true,
			errMsg:  "file test.conf has invalid Base64 content",
		},
		{
			name: "file with invalid mode (negative)",
			bundle: &protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    "test.conf",
						Content: base64.StdEncoding.EncodeToString([]byte("content")),
						Mode:    -1,
					},
				},
			},
			wantErr: true,
			errMsg:  "file test.conf has invalid mode",
		},
		{
			name: "file with invalid mode (too high)",
			bundle: &protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    "test.conf",
						Content: base64.StdEncoding.EncodeToString([]byte("content")),
						Mode:    0o1000,
					},
				},
			},
			wantErr: true,
			errMsg:  "file test.conf has invalid mode",
		},
		{
			name: "bundle exceeds file count",
			bundle: func() *protocol.ConfigBundle {
				files := make([]protocol.ConfigFile, MaxFileCount+1)
				for i := range files {
					files[i] = protocol.ConfigFile{
						Path:    "file" + string(rune(i)) + ".conf",
						Content: base64.StdEncoding.EncodeToString([]byte("content")),
						Mode:    0o644,
					}
				}
				return &protocol.ConfigBundle{Files: files}
			}(),
			wantErr: true,
			errMsg:  "bundle contains 101 files, maximum is 100",
		},
		{
			name: "bundle exceeds size limit",
			bundle: &protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    "large.bin",
						Content: base64.StdEncoding.EncodeToString(make([]byte, MaxBundleSize+1)),
						Mode:    0o644,
					},
				},
			},
			wantErr: true,
			errMsg:  "bundle exceeds maximum size",
		},
		{
			name: "valid bundle with multiple files",
			bundle: &protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    "file1.conf",
						Content: base64.StdEncoding.EncodeToString([]byte("content 1")),
						Mode:    0o644,
					},
					{
						Path:    "file2.conf",
						Content: base64.StdEncoding.EncodeToString([]byte("content 2")),
						Mode:    0o600,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBundle(tt.bundle)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDecodeFileContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []byte
		wantErr bool
	}{
		{
			name:    "valid Base64",
			content: base64.StdEncoding.EncodeToString([]byte("test content")),
			want:    []byte("test content"),
			wantErr: false,
		},
		{
			name:    "empty string",
			content: "",
			want:    []byte{},
			wantErr: false,
		},
		{
			name:    "invalid Base64",
			content: "not-valid-base64!!!",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "binary content",
			content: base64.StdEncoding.EncodeToString([]byte{0x00, 0x01, 0x02, 0xff}),
			want:    []byte{0x00, 0x01, 0x02, 0xff},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeFileContent(tt.content)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to decode Base64")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestValidateBundle_SizeCalculation(t *testing.T) {
	// Create a bundle with multiple files that total just under the limit
	fileSize := 1024 * 1024 // 1MB each
	numFiles := 9           // 9 files = 9MB, under 10MB limit

	files := make([]protocol.ConfigFile, numFiles)
	for i := 0; i < numFiles; i++ {
		content := strings.Repeat("a", fileSize)
		files[i] = protocol.ConfigFile{
			Path:    "file" + string(rune(i)) + ".bin",
			Content: base64.StdEncoding.EncodeToString([]byte(content)),
			Mode:    0o644,
		}
	}

	bundle := &protocol.ConfigBundle{Files: files}
	err := ValidateBundle(bundle)
	assert.NoError(t, err, "bundle under size limit should be valid")
}

func TestValidateBundle_ExactFileCountLimit(t *testing.T) {
	// Create bundle with exactly MaxFileCount files
	files := make([]protocol.ConfigFile, MaxFileCount)
	for i := 0; i < MaxFileCount; i++ {
		files[i] = protocol.ConfigFile{
			Path:    "file" + string(rune(i)) + ".conf",
			Content: base64.StdEncoding.EncodeToString([]byte("content")),
			Mode:    0o644,
		}
	}

	bundle := &protocol.ConfigBundle{Files: files}
	err := ValidateBundle(bundle)
	assert.NoError(t, err, "bundle with exactly MaxFileCount files should be valid")
}

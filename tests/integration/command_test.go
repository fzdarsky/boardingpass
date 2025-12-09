package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/internal/command"
	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCommandHandler_POST_Success(t *testing.T) {
	tests := []struct {
		name         string
		commandID    string
		wantExitCode int
		wantStdout   string
		wantStderr   string
	}{
		{
			name:         "execute echo command",
			commandID:    "echo-test",
			wantExitCode: 0,
			wantStdout:   "hello world\n",
			wantStderr:   "",
		},
		{
			name:         "execute true command",
			commandID:    "true-test",
			wantExitCode: 0,
			wantStdout:   "",
			wantStderr:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockExecutor := command.NewMockCommandExecutor(ctrl)

			// Set expectation for Execute call
			mockExecutor.EXPECT().
				Execute(gomock.Any(), gomock.Any(), true).
				Return(&protocol.CommandResponse{
					ExitCode: tt.wantExitCode,
					Stdout:   tt.wantStdout,
					Stderr:   tt.wantStderr,
				}, nil).
				Times(1)

			// Create test configuration
			testConfig := &config.Config{
				Commands: []config.CommandDefinition{
					{
						ID:   "echo-test",
						Path: "/bin/echo",
						Args: []string{"hello", "world"},
					},
					{
						ID:   "true-test",
						Path: "/bin/true",
						Args: []string{},
					},
				},
			}

			logger := logging.New(logging.LevelInfo, logging.FormatJSON)
			handler, err := handlers.NewCommandHandlerWithExecutor(testConfig, mockExecutor, logger)
			require.NoError(t, err)
			require.NotNil(t, handler)

			reqBody := protocol.CommandRequest{
				ID: tt.commandID,
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response protocol.CommandResponse
			err = json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)

			assert.Equal(t, tt.wantExitCode, response.ExitCode)
			assert.Equal(t, tt.wantStdout, response.Stdout)
			assert.Equal(t, tt.wantStderr, response.Stderr)
		})
	}
}

func TestCommandHandler_POST_CommandNotInAllowList(t *testing.T) {
	testConfig := &config.Config{
		Commands: []config.CommandDefinition{
			{
				ID:   "echo-test",
				Path: "/bin/echo",
				Args: []string{"hello"},
			},
		},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler, err := handlers.NewCommandHandler(testConfig, logger)
	require.NoError(t, err)

	reqBody := protocol.CommandRequest{
		ID: "unknown-command",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "command_not_allowed", response["error"])
	assert.Contains(t, response["message"], "unknown-command")
	assert.Contains(t, response["message"], "not in the allow-list")
}

func TestCommandHandler_POST_InvalidJSON(t *testing.T) {
	testConfig := &config.Config{
		Commands: []config.CommandDefinition{
			{
				ID:   "echo-test",
				Path: "/bin/echo",
				Args: []string{"hello"},
			},
		},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler, err := handlers.NewCommandHandler(testConfig, logger)
	require.NoError(t, err)

	// Invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestCommandHandler_POST_MethodNotAllowed(t *testing.T) {
	testConfig := &config.Config{
		Commands: []config.CommandDefinition{
			{
				ID:   "echo-test",
				Path: "/bin/echo",
				Args: []string{"hello"},
			},
		},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler, err := handlers.NewCommandHandler(testConfig, logger)
	require.NoError(t, err)

	// Try GET instead of POST
	req := httptest.NewRequest(http.MethodGet, "/command", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestCommandHandler_POST_NonZeroExitCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExecutor := command.NewMockCommandExecutor(ctrl)

	// Mock should return exit code 1
	mockExecutor.EXPECT().
		Execute(gomock.Any(), gomock.Any(), true).
		Return(&protocol.CommandResponse{
			ExitCode: 1,
			Stdout:   "",
			Stderr:   "",
		}, nil).
		Times(1)

	testConfig := &config.Config{
		Commands: []config.CommandDefinition{
			{
				ID:   "false-test",
				Path: "/bin/false",
				Args: []string{},
			},
		},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler, err := handlers.NewCommandHandlerWithExecutor(testConfig, mockExecutor, logger)
	require.NoError(t, err)

	reqBody := protocol.CommandRequest{
		ID: "false-test",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Command executed successfully (HTTP 200), but exit code is non-zero
	assert.Equal(t, http.StatusOK, w.Code)

	var response protocol.CommandResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.ExitCode)
}

func TestCommandHandler_POST_StderrCapture(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExecutor := command.NewMockCommandExecutor(ctrl)

	// Mock should capture stderr
	mockExecutor.EXPECT().
		Execute(gomock.Any(), gomock.Any(), true).
		Return(&protocol.CommandResponse{
			ExitCode: 0,
			Stdout:   "",
			Stderr:   "error\n",
		}, nil).
		Times(1)

	testConfig := &config.Config{
		Commands: []config.CommandDefinition{
			{
				ID:   "stderr-test",
				Path: "/bin/sh",
				Args: []string{"-c", "echo error >&2"},
			},
		},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler, err := handlers.NewCommandHandlerWithExecutor(testConfig, mockExecutor, logger)
	require.NoError(t, err)

	reqBody := protocol.CommandRequest{
		ID: "stderr-test",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response protocol.CommandResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 0, response.ExitCode)
	assert.Equal(t, "error\n", response.Stderr)
}

func TestNewCommandHandler_EmptyCommandList(t *testing.T) {
	testConfig := &config.Config{
		Commands: []config.CommandDefinition{},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler, err := handlers.NewCommandHandler(testConfig, logger)

	assert.Error(t, err)
	assert.Nil(t, handler)
	assert.Contains(t, err.Error(), "command allow-list cannot be empty")
}

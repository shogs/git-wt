package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Session represents worktree session information
type Session struct {
	Branch     string
	Created    time.Time
	BaseBranch string
	Task       string
}

// getSessionPath returns the path to the session file for a worktree
func getSessionPath(worktreePath string) string {
	return filepath.Join(worktreePath, ".git-wt-session")
}

// loadSession loads session information from a worktree
func loadSession(worktreePath string) (*Session, error) {
	sessionPath := getSessionPath(worktreePath)
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No session file
		}
		return nil, err
	}

	session := &Session{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "branch":
			session.Branch = value
		case "created":
			if t, err := time.Parse(time.RFC3339, value); err == nil {
				session.Created = t
			}
		case "base_branch":
			session.BaseBranch = value
		case "task":
			session.Task = value
		}
	}

	return session, nil
}

// saveSession saves session information to a worktree
func saveSession(worktreePath string, session *Session) error {
	sessionPath := getSessionPath(worktreePath)

	content := fmt.Sprintf(`branch=%s
created=%s
base_branch=%s
task=%s
`,
		session.Branch,
		session.Created.Format(time.RFC3339),
		session.BaseBranch,
		session.Task,
	)

	return os.WriteFile(sessionPath, []byte(content), 0644)
}

// deleteSession removes the session file from a worktree
func deleteSession(worktreePath string) error {
	sessionPath := getSessionPath(worktreePath)
	err := os.Remove(sessionPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

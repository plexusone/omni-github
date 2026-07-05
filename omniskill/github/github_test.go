// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"testing"

	"github.com/plexusone/omniskill/skill"
)

func TestSkill_Name(t *testing.T) {
	s := New(Config{})
	if s.Name() != "github" {
		t.Errorf("Name() = %q, want %q", s.Name(), "github")
	}
}

func TestSkill_Description(t *testing.T) {
	s := New(Config{})
	if s.Description() == "" {
		t.Error("Description() should not be empty")
	}
}

func TestSkill_Tools(t *testing.T) {
	s := New(Config{Token: "test"})

	tools := s.Tools()
	if len(tools) != 10 {
		t.Errorf("Tools() returned %d tools, want 10", len(tools))
	}

	expectedTools := []string{
		"list_issues",
		"get_issue",
		"create_issue",
		"update_issue",
		"add_issue_comment",
		"list_pull_requests",
		"get_pull_request",
		"add_pull_request_comment",
		"search_code",
		"search_issues",
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("missing tool %q", expected)
		}
	}
}

func TestSkill_Init_NoToken(t *testing.T) {
	s := New(Config{})
	err := s.Init(context.Background())
	if err == nil {
		t.Error("Init() should error without token")
	}
}

func TestSkill_ImplementsInterface(t *testing.T) {
	var _ skill.Skill = (*Skill)(nil)
}

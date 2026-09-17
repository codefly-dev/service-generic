package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The agent-context standard these tests enforce is fleet-wide:
// obin-ai/handbook#68. The budgets are not stylistic — an over-long root file
// measurably reduces adherence, and a skill's description is the only text an
// agent sees before deciding whether to load it.
const (
	agentContextFile         = "AGENTS.md"
	agentContextMaxLines     = 200
	skillsDir                = ".claude/skills"
	skillBodyMaxLines        = 500
	skillDescriptionMaxChars = 1024
)

var skillNamePattern = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

func TestAgentContextFileWithinLengthBudget(t *testing.T) {
	content, err := os.ReadFile(agentContextFile)
	if err != nil {
		t.Fatalf("read %s: %v", agentContextFile, err)
	}
	if lines := countLines(string(content)); lines > agentContextMaxLines {
		t.Fatalf("%s is %d lines, over the %d-line cap; move depth into %s or a nested AGENTS.md",
			agentContextFile, lines, agentContextMaxLines, skillsDir)
	}
}

// TestClaudeMemoryFileIsAPointer keeps a single canonical source. A CLAUDE.md
// that restates AGENTS.md drifts from it; the standard's remedy is a pointer.
func TestClaudeMemoryFileIsAPointer(t *testing.T) {
	content, err := os.ReadFile("CLAUDE.md")
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	if strings.TrimSpace(string(content)) != "@"+agentContextFile {
		t.Fatalf("CLAUDE.md must be exactly %q, got %q", "@"+agentContextFile, string(content))
	}
}

func TestSkillsAreLoadable(t *testing.T) {
	for _, name := range skillNames(t) {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(skillsDir, name, "SKILL.md")
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("every skill directory needs a SKILL.md: %v", err)
			}
			fields, body, err := parseFrontmatter(string(content))
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			if fields["name"] != name {
				t.Errorf("%s: name = %q, must match its directory %q", path, fields["name"], name)
			}
			if !skillNamePattern.MatchString(fields["name"]) {
				t.Errorf("%s: name %q must be lowercase letters, digits, or hyphens, at most 64 characters", path, fields["name"])
			}
			description := strings.TrimSpace(fields["description"])
			if description == "" {
				t.Errorf("%s: description is required — it is the only text loaded before the skill triggers", path)
			}
			if len(description) > skillDescriptionMaxChars {
				t.Errorf("%s: description is %d characters, over the %d cap", path, len(description), skillDescriptionMaxChars)
			}
			if lines := countLines(body); lines > skillBodyMaxLines {
				t.Errorf("%s: body is %d lines, over the %d cap; move detail into a reference file the skill reads on demand",
					path, lines, skillBodyMaxLines)
			}
		})
	}
}

// TestSkillsAreReachableFromAgentContext closes the progressive-disclosure loop:
// a skill the root file never names is one an agent has no reason to look for.
func TestSkillsAreReachableFromAgentContext(t *testing.T) {
	content, err := os.ReadFile(agentContextFile)
	if err != nil {
		t.Fatalf("read %s: %v", agentContextFile, err)
	}
	for _, name := range skillNames(t) {
		if !strings.Contains(string(content), name) {
			t.Errorf("skill %q is not named in %s", name, agentContextFile)
		}
	}
}

func skillNames(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(skillsDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read %s: %v", skillsDir, err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}

// parseFrontmatter reads the leading YAML block's top-level scalar keys. The
// contract requires only name and description, both single-line, so this reads
// the block directly rather than promoting a YAML dependency for two fields.
func parseFrontmatter(content string) (map[string]string, string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, "", errFrontmatter("must open with a --- frontmatter delimiter on the first line")
	}
	fields := map[string]string{}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return fields, strings.Join(lines[i+1:], "\n"), nil
		}
		key, value, found := strings.Cut(lines[i], ":")
		if found && !strings.HasPrefix(key, " ") {
			fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return nil, "", errFrontmatter("frontmatter is never closed by a --- delimiter")
}

type errFrontmatter string

func (e errFrontmatter) Error() string { return string(e) }

func countLines(content string) int {
	trimmed := strings.Trim(content, "\n")
	if trimmed == "" {
		return 0
	}
	return strings.Count(trimmed, "\n") + 1
}

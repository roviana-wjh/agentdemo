package blog_tools

import (
	"context"
	"fmt"

	"go-agent/tool/skill"
	"go-agent/tool/trace"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

var globalSkillStore skill.SkillStore

// SetSkillStore sets the global skill store for skill_view tool.
func SetSkillStore(store skill.SkillStore) {
	globalSkillStore = store
}

type SkillViewInput struct {
	Name string `json:"name" jsonschema:"required,description=Skill name to view details"`
}

type SkillViewResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Procedure   string `json:"procedure"`
	ToolsUsed   string `json:"tools_used"`
}

func newSkillViewTool() (tool.BaseTool, error) {
	if globalSkillStore == nil {
		return nil, fmt.Errorf("skill store not initialized")
	}

	return utils.InferTool(
		"skill_view",
		"View detailed procedure of a learned skill. Use this when you need to know how to execute a specific skill.",
		func(ctx context.Context, input SkillViewInput) (SkillViewResult, error) {
			s, err := globalSkillStore.Get(ctx, input.Name)
			if err != nil {
				trace.LogSkillUsage(input.Name, false)
				return SkillViewResult{}, fmt.Errorf("skill not found: %w", err)
			}
			// Increment usage count asynchronously
			go func() {
				if incrErr := globalSkillStore.IncrUsage(context.Background(), input.Name); incrErr == nil {
					trace.LogSkillUsage(input.Name, true)
				} else {
					trace.LogSkillUsage(input.Name, false)
				}
			}()
			return SkillViewResult{
				Name:        s.Name,
				Description: s.Description,
				Procedure:   s.Procedure,
				ToolsUsed:   s.ToolsUsed,
			}, nil
		},
	)
}

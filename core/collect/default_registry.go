package collect

import (
	"fmt"

	"github.com/Clyra-AI/axym/core/collect/dbt"
	"github.com/Clyra-AI/axym/core/collect/githubactions"
	"github.com/Clyra-AI/axym/core/collect/gitmeta"
	"github.com/Clyra-AI/axym/core/collect/governanceevent"
	"github.com/Clyra-AI/axym/core/collect/llmapi"
	"github.com/Clyra-AI/axym/core/collect/mcp"
	"github.com/Clyra-AI/axym/core/collect/plugin"
	"github.com/Clyra-AI/axym/core/collect/snowflake"
	"github.com/Clyra-AI/axym/core/collect/webhook"
	"github.com/Clyra-AI/axym/core/collector"
)

func BuildRegistry(req collector.Request) (*collector.Registry, error) {
	registry := collector.NewRegistry()
	base := []collector.Collector{
		mcp.Collector{},
		llmapi.Collector{},
		webhook.Collector{},
		githubactions.Collector{},
		gitmeta.Collector{},
		dbt.Collector{},
		snowflake.Collector{},
		governanceevent.Collector{},
	}
	for _, item := range base {
		if err := registry.Register(item); err != nil {
			return nil, err
		}
	}
	for i, cmd := range req.PluginCommands {
		name := fmt.Sprintf("plugin:%02d", i+1)
		if err := registry.Register(plugin.Collector{Command: cmd, Timeout: req.PluginTimeout, NameID: name}); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

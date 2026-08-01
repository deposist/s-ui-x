package core

import (
	"context"

	sb "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/option"
)

// registryContext carries every official registry needed to decode the
// discriminated option unions. Rule validation and full config validation use
// the same context so panel checks cannot drift from runtime parsing.
func registryContext(ctx context.Context) context.Context {
	return sb.Context(ctx, InboundRegistry(), OutboundRegistry(), EndpointRegistry(), DNSTransportRegistry(), ServiceRegistry())
}

// ValidateConfig builds a sing-box instance from the supplied config without
// starting its lifecycle. It catches parse and construction failures while
// avoiding listener binds, outbound dials, and cache/ruleset downloads.
func ValidateConfig(sbConfig []byte) error {
	var opt option.Options
	ctx := registryContext(context.Background())
	if err := opt.UnmarshalJSONContext(ctx, sbConfig); err != nil {
		return err
	}
	instance, err := NewBox(Options{
		Context: ctx,
		Options: opt,
	})
	if err != nil {
		return err
	}
	return instance.Close()
}

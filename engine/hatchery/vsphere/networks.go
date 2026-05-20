package vsphere

import (
	"context"
	"fmt"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

// initNetworks builds the list of available networks from configuration.
// It supports both the new Networks config and the legacy single IPRange/Gateway/SubnetMask fields.
func (h *HatcheryVSphere) initNetworks(ctx context.Context) error {
	h.availableNetworks = nil
	h.availableIPAddresses = nil

	// Build the effective list of network configs.
	// New-style Networks takes precedence; if empty, fall back to legacy fields.
	networks := h.Config.Networks
	if len(networks) == 0 && h.Config.IPRange != "" {
		networks = []NetworkConfig{
			{
				IPRange:    h.Config.IPRange,
				Gateway:    h.Config.Gateway,
				SubnetMask: h.Config.SubnetMask,
			},
		}
	}

	for i, netCfg := range networks {
		if netCfg.IPRange == "" {
			continue
		}
		if netCfg.Gateway == "" {
			return fmt.Errorf("networks[%d]: gateway is required", i)
		}
		if netCfg.SubnetMask == "" {
			return fmt.Errorf("networks[%d]: subnetMask is required", i)
		}

		ips, err := sdk.IPinRanges(ctx, netCfg.IPRange)
		if err != nil {
			return fmt.Errorf("networks[%d] ip-range error: %v", i, err)
		}

		h.availableNetworks = append(h.availableNetworks, availableNetwork{
			config:      netCfg,
			ipAddresses: ips,
		})
		h.availableIPAddresses = append(h.availableIPAddresses, ips...)
	}

	if len(h.availableNetworks) > 0 {
		totalIPs := len(h.availableIPAddresses)
		log.Info(ctx, "network configuration: %d network(s), %d total IPs available", len(h.availableNetworks), totalIPs)
		for i, net := range h.availableNetworks {
			log.Info(ctx, "  network[%d] gw=%s: %d IPs (mask=%s)",
				i, net.config.Gateway, len(net.ipAddresses), net.config.SubnetMask)
		}
	}

	return nil
}

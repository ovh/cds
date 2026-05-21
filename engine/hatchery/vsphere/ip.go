package vsphere

import (
	"context"
	"errors"
	"time"

	"github.com/ovh/cds/sdk"
)

// findAvailableIP looks for the first free IP across all configured networks
// and returns the IP along with its associated network configuration.
func (h *HatcheryVSphere) findAvailableIP(ctx context.Context) (ipResult, error) {
	h.IpAddressesMutex.Lock()
	defer h.IpAddressesMutex.Unlock()

	srvs := h.getVirtualMachines(ctx)

	var usedIPAddresses = make(map[string]struct{}, len(srvs))
	for _, s := range srvs {
		var annots = getVirtualMachineCDSAnnotation(ctx, s)
		if annots != nil {
			var ip = annots.IPAddress
			if ip != "" {
				usedIPAddresses[ip] = struct{}{}
			}
		}
		if s.Guest == nil {
			continue
		}
		for _, n := range s.Guest.Net {
			for _, ip := range n.IpAddress {
				usedIPAddresses[ip] = struct{}{}
				// If the IP Address is know as a reservedIPAddress, remove it
				if sdk.IsInArray(ip, h.reservedIPAddresses) {
					h.reservedIPAddresses = sdk.DeleteFromArray(h.reservedIPAddresses, ip)
				}
			}
		}
	}

	for _, network := range h.availableNetworks {
		for _, ip := range network.ipAddresses {
			_, isUsed := usedIPAddresses[ip]
			isReserved := sdk.IsInArray(ip, h.reservedIPAddresses)
			if !isUsed && !isReserved {
				return ipResult{
					ip:         ip,
					gateway:    network.config.Gateway,
					subnetMask: network.config.SubnetMask,
				}, nil
			}
		}
	}

	return ipResult{}, sdk.WithStack(errors.New("no IP address available"))
}

func (h *HatcheryVSphere) reserveIPAddress(ctx context.Context, ip string) error {
	h.IpAddressesMutex.Lock()
	if sdk.IsInArray(ip, h.reservedIPAddresses) {
		return sdk.WithStack(errors.New("address already reserved"))
	}
	h.reservedIPAddresses = append(h.reservedIPAddresses, ip)
	h.IpAddressesMutex.Unlock()

	go func() {
		time.Sleep(5 * time.Minute)
		h.IpAddressesMutex.Lock()
		h.reservedIPAddresses = sdk.DeleteFromArray(h.reservedIPAddresses, ip)
		h.IpAddressesMutex.Unlock()
	}()

	return nil
}

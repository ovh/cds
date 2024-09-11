package vsphere

import (
	"context"
	"errors"
	"time"

	"github.com/ovh/cds/sdk"
)

// For each IPs in the range, look for the first free ones
func (h *HatcheryVSphere) findAvailableIP(ctx context.Context) (string, error) {
	h.IpAddressesMutex.Lock()
	defer h.IpAddressesMutex.Unlock()

	srvs := h.getVirtualMachines(ctx)

	var usedIPAddresses = make(map[string]struct{}, len(srvs))
	for _, s := range srvs {
		var annots = getVirtualMachineCDSAnnotation(s)
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

	for _, ip := range h.availableIPAddresses {
		_, isUsed := usedIPAddresses[ip]
		isReserved := sdk.IsInArray(ip, h.reservedIPAddresses)
		if !isUsed && !isReserved {
			return ip, nil
		}
	}

	return "", sdk.WithStack(errors.New("no IP address available"))
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

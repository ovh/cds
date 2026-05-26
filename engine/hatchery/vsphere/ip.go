package vsphere

import (
	"context"
	"errors"
	"time"

	"github.com/vmware/govmomi/vim25/mo"

	"github.com/ovh/cds/sdk"
)

// getUsedIPs builds the set of IP addresses currently in use across all VMs.
// It checks both the CDS annotation (IP assigned at clone time, before the VM
// actually uses it) and the guest network info (IP visible on the running VM).
// The annotation lookup is needed because a newly cloned VM may not yet have
// the IP visible in guest tools, but the IP is already allocated.
func (h *HatcheryVSphere) getUsedIPs(ctx context.Context, srvs []mo.VirtualMachine) map[string]struct{} {
	usedIPs := make(map[string]struct{}, len(srvs))
	for _, s := range srvs {
		annot := getVirtualMachineCDSAnnotation(ctx, s)
		if annot != nil && annot.IPAddress != "" {
			usedIPs[annot.IPAddress] = struct{}{}
		}
		if s.Guest == nil {
			continue
		}
		for _, n := range s.Guest.Net {
			for _, ip := range n.IpAddress {
				usedIPs[ip] = struct{}{}
			}
		}
	}
	return usedIPs
}

// findAvailableIP looks for the first free IP across all configured networks
// and returns the IP along with its associated network configuration.
func (h *HatcheryVSphere) findAvailableIP(ctx context.Context) (ipResult, error) {
	h.IpAddressesMutex.Lock()
	defer h.IpAddressesMutex.Unlock()

	srvs := h.getVirtualMachines(ctx)
	usedIPs := h.getUsedIPs(ctx, srvs)

	// If an IP appears in guest network, it's confirmed in use — clean up stale reservations
	for ip := range usedIPs {
		if sdk.IsInArray(ip, h.reservedIPAddresses) {
			h.reservedIPAddresses = sdk.DeleteFromArray(h.reservedIPAddresses, ip)
		}
	}

	for _, network := range h.availableNetworks {
		for _, ip := range network.ipAddresses {
			_, isUsed := usedIPs[ip]
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

// countAvailableIPs returns the number of IPs that are currently free (not used
// by any VM and not reserved). Used by provisioning to cap tasks to the actual
// IP budget and avoid submitting work that would fail at clone time.
func (h *HatcheryVSphere) countAvailableIPs(ctx context.Context) int {
	h.IpAddressesMutex.Lock()
	defer h.IpAddressesMutex.Unlock()

	srvs := h.getVirtualMachines(ctx)
	usedIPs := h.getUsedIPs(ctx, srvs)

	count := 0
	for _, network := range h.availableNetworks {
		for _, ip := range network.ipAddresses {
			_, isUsed := usedIPs[ip]
			isReserved := sdk.IsInArray(ip, h.reservedIPAddresses)
			if !isUsed && !isReserved {
				count++
			}
		}
	}
	return count
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

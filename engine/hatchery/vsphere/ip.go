package vsphere

import (
	"context"
	"strconv"
	"strings"

	"github.com/vmware/govmomi/vim25/mo"

	"github.com/ovh/cds/sdk/namesgenerator"
)

// provisionIPNamePrefix marks a provision VM name that encodes its assigned IP,
// e.g. "provision-v2-ip-10-0-0-5-<random>". Encoding the IP in the name makes the
// reserved IP visible in the vSphere inventory from the moment the clone starts —
// before the annotation is populated — and survives a hatchery restart. The IP is
// also stored in the annotation (see prepareCloneSpec) so older provisions
// (without this name) and a rolled-back binary keep working.
const provisionIPNamePrefix = "provision-v2-ip-"

// provisionName builds the name for a new provision VM. When an IP is assigned it
// is encoded into the name; otherwise (DHCP / no IP range) a plain name is used.
func provisionName(ip *ipResult) string {
	if ip == nil || ip.ip == "" {
		return namesgenerator.GenerateWorkerName("provision-v2")
	}
	return namesgenerator.GenerateWorkerName(provisionIPNamePrefix + strings.ReplaceAll(ip.ip, ".", "-"))
}

// ipFromProvisionName extracts the IP encoded in a provision VM name, if present.
func ipFromProvisionName(name string) (string, bool) {
	if !strings.HasPrefix(name, provisionIPNamePrefix) {
		return "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(name, provisionIPNamePrefix), "-", 5)
	if len(parts) < 4 {
		return "", false
	}
	for _, octet := range parts[:4] {
		n, err := strconv.Atoi(octet)
		if err != nil || n < 0 || n > 255 {
			return "", false
		}
	}
	return strings.Join(parts[:4], "."), true
}

// getUsedIPs builds the set of IP addresses currently in use across all VMs. An
// IP is "used" if it is encoded in a provision VM name (visible during the clone,
// before the annotation exists), recorded in the CDS annotation (set at clone
// time, the compatibility anchor for old provisions and claimed workers), or seen
// on the running guest's network.
func (h *HatcheryVSphere) getUsedIPs(ctx context.Context, srvs []mo.VirtualMachine) map[string]struct{} {
	usedIPs := make(map[string]struct{}, len(srvs))
	for _, s := range srvs {
		if ip, ok := ipFromProvisionName(s.Name); ok {
			usedIPs[ip] = struct{}{}
		}
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

// pickFreeIP returns the first configured IP that is not in the given used set,
// scanning networks in config order. It is pure (no shared state): the caller
// (provisioningV2, a single goroutine) adds each picked IP to the used set so the
// next pick in the same pass returns a distinct address.
func (h *HatcheryVSphere) pickFreeIP(used map[string]struct{}) (ipResult, bool) {
	for _, network := range h.availableNetworks {
		for _, ip := range network.ipAddresses {
			if _, taken := used[ip]; !taken {
				return ipResult{
					ip:         ip,
					gateway:    network.config.Gateway,
					subnetMask: network.config.SubnetMask,
				}, true
			}
		}
	}
	return ipResult{}, false
}

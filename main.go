package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
)

type lanNet struct {
	Name          string
	NumDevices    int
	Size          int
	Base          uint32
	RouterIfaceIP uint32
}

func ipToU32(ip net.IP) uint32 {
	return binary.BigEndian.Uint32(ip.To4())
}

func u32ToIP(n uint32) net.IP {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, n)
	return net.IP(b)
}

func nextPow2GE(n int) int {
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

func sizeToMask(size int) net.IP {
	hostBits := 0
	for s := size; s > 1; s >>= 1 {
		hostBits++
	}
	mask := uint32(0xFFFFFFFF) << uint32(hostBits)
	return u32ToIP(mask)
}

type interval struct{ lo, hi uint32 }

// allocate performs VLSM largest-first allocation. Returns bases in input order.
func allocate(start, end uint32, sizes []int) ([]uint32, error) {
	type item struct {
		idx  int
		size int
		base uint32
	}
	items := make([]item, len(sizes))
	for i, s := range sizes {
		items[i] = item{idx: i, size: s}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].size > items[j].size })

	free := []interval{{start, end}}

	for k := range items {
		s := uint32(items[k].size)
		placed := false
		for j := 0; j < len(free); j++ {
			iv := free[j]
			aligned := ((iv.lo + s - 1) / s) * s
			if aligned+s-1 <= iv.hi && aligned >= iv.lo {
				items[k].base = aligned
				newFree := make([]interval, 0, len(free)+1)
				newFree = append(newFree, free[:j]...)
				if aligned > iv.lo {
					newFree = append(newFree, interval{iv.lo, aligned - 1})
				}
				if aligned+s-1 < iv.hi {
					newFree = append(newFree, interval{aligned + s, iv.hi})
				}
				newFree = append(newFree, free[j+1:]...)
				free = newFree
				placed = true
				break
			}
		}
		if !placed {
			return nil, fmt.Errorf("insufficient address space for subnet of size %d", s)
		}
	}

	result := make([]uint32, len(sizes))
	for _, it := range items {
		result[it.idx] = it.base
	}
	return result, nil
}

func parseArgs(args []string) (uint32, uint32, []lanNet, error) {
	var startStr, endStr string
	nNets := -1
	var lans []lanNet
	var pendingName string
	havePending := false

	i := 0
	for i < len(args) {
		switch args[i] {
		case "-ip":
			if i+2 >= len(args) {
				return 0, 0, nil, fmt.Errorf("-ip requires <start> <end>")
			}
			startStr, endStr = args[i+1], args[i+2]
			i += 3
		case "-r":
			if i+1 >= len(args) {
				return 0, 0, nil, fmt.Errorf("-r requires a number")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return 0, 0, nil, fmt.Errorf("invalid -r value %q: %v", args[i+1], err)
			}
			nNets = n
			i += 2
		case "-name":
			if i+1 >= len(args) {
				return 0, 0, nil, fmt.Errorf("-name requires a value")
			}
			if havePending {
				return 0, 0, nil, fmt.Errorf("two -name in a row, expected -dev for %q", pendingName)
			}
			pendingName = args[i+1]
			havePending = true
			i += 2
		case "-dev":
			if i+1 >= len(args) {
				return 0, 0, nil, fmt.Errorf("-dev requires a number")
			}
			if !havePending {
				return 0, 0, nil, fmt.Errorf("-dev without preceding -name")
			}
			d, err := strconv.Atoi(args[i+1])
			if err != nil {
				return 0, 0, nil, fmt.Errorf("invalid -dev value %q: %v", args[i+1], err)
			}
			if d < 1 {
				return 0, 0, nil, fmt.Errorf("-dev must be >= 1")
			}
			lans = append(lans, lanNet{Name: pendingName, NumDevices: d})
			havePending = false
			i += 2
		default:
			return 0, 0, nil, fmt.Errorf("unknown argument %q", args[i])
		}
	}

	if startStr == "" || endStr == "" {
		return 0, 0, nil, fmt.Errorf("missing -ip <start> <end>")
	}
	sIP := net.ParseIP(startStr)
	eIP := net.ParseIP(endStr)
	if sIP == nil || eIP == nil || sIP.To4() == nil || eIP.To4() == nil {
		return 0, 0, nil, fmt.Errorf("invalid IPv4 in -ip range")
	}
	start := ipToU32(sIP)
	end := ipToU32(eIP)
	if end < start {
		return 0, 0, nil, fmt.Errorf("end IP < start IP")
	}
	if nNets < 1 {
		return 0, 0, nil, fmt.Errorf("missing or invalid -r")
	}
	if nNets != len(lans) {
		return 0, 0, nil, fmt.Errorf("-r says %d networks but got %d -name/-dev pairs", nNets, len(lans))
	}
	if havePending {
		return 0, 0, nil, fmt.Errorf("trailing -name %q without -dev", pendingName)
	}
	return start, end, lans, nil
}

func printRange(w *os.File, label string, lo, hi uint32) {
	switch {
	case lo > hi:
		fmt.Fprintf(w, "%s: -\n", label)
	case lo == hi:
		fmt.Fprintf(w, "%s: %s\n", label, u32ToIP(lo))
	default:
		fmt.Fprintf(w, "%s: %s -> %s\n", label, u32ToIP(lo), u32ToIP(hi))
	}
}

func run(args []string, w *os.File) error {
	start, end, lans, err := parseArgs(args)
	if err != nil {
		return err
	}

	for i := range lans {
		lans[i].Size = nextPow2GE(lans[i].NumDevices + 2)
	}

	hasRouterNet := len(lans) >= 2
	var routerSize int
	if hasRouterNet {
		routerSize = nextPow2GE(len(lans) + 2)
	}

	sizes := make([]int, 0, len(lans)+1)
	for _, l := range lans {
		sizes = append(sizes, l.Size)
	}
	if hasRouterNet {
		sizes = append(sizes, routerSize)
	}

	bases, err := allocate(start, end, sizes)
	if err != nil {
		return err
	}
	for i := range lans {
		lans[i].Base = bases[i]
	}
	var routerBase uint32
	if hasRouterNet {
		routerBase = bases[len(lans)]
		for i := range lans {
			lans[i].RouterIfaceIP = routerBase + 1 + uint32(i)
		}
	}

	for _, l := range lans {
		fmt.Fprintf(w, "=== %s ===\n", l.Name)
		fmt.Fprintf(w, "Size: %d\n", l.Size)
		ar := l.Base
		ab := l.Base + uint32(l.Size) - 1
		gw := l.Base + 1
		fmt.Fprintf(w, "AR: %s\n", u32ToIP(ar))
		fmt.Fprintf(w, "AB: %s\n", u32ToIP(ab))

		// Devices include the router (gateway). Non-gateway hosts: base+2 .. base+NumDevices
		hostsLo := l.Base + 2
		hostsHi := l.Base + uint32(l.NumDevices)
		if l.NumDevices < 2 {
			fmt.Fprintln(w, "Allocated IP's: -")
		} else {
			printRange(w, "Allocated IP's", hostsLo, hostsHi)
		}

		unallocLo := hostsHi + 1
		if l.NumDevices < 2 {
			unallocLo = gw + 1
		}
		unallocHi := ab - 1
		printRange(w, "Unallocated IP's", unallocLo, unallocHi)

		fmt.Fprintf(w, "Mask: %s\n", sizeToMask(l.Size))
		fmt.Fprintf(w, "Gateway %s\n", u32ToIP(gw))
		fmt.Fprintln(w)
	}

	if hasRouterNet {
		fmt.Fprintln(w, "=== ROUTER NETWORK ===")
		fmt.Fprintf(w, "Size: %d\n", routerSize)
		rAR := routerBase
		rAB := routerBase + uint32(routerSize) - 1
		fmt.Fprintf(w, "AR: %s\n", u32ToIP(rAR))
		fmt.Fprintf(w, "AB: %s\n", u32ToIP(rAB))
		firstR := routerBase + 1
		lastR := routerBase + uint32(len(lans))
		printRange(w, "Allocated IP's", firstR, lastR)
		fmt.Fprintf(w, "Mask: %s\n", sizeToMask(routerSize))
		fmt.Fprintln(w)

		for i, l := range lans {
			fmt.Fprintf(w, "=== ROUTING TABLE FOR %s's Router ===\n", l.Name)
			for j, other := range lans {
				if i == j {
					continue
				}
				fmt.Fprintf(w, "Network: %s\n", u32ToIP(other.Base))
				fmt.Fprintf(w, "Mask: %s\n", sizeToMask(other.Size))
				fmt.Fprintf(w, "Next-Hop : %s\n", u32ToIP(other.RouterIfaceIP))
			}
			fmt.Fprintln(w)
		}
	}

	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

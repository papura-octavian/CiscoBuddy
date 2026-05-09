# CiscoBuddy

A CLI tool written in Go that solves VLSM subnetting problems: it allocates subnets from an IP range, computes the masks, and generates the routing tables for the routers connecting the networks.

Covers parts 1 and 2 of typical DHCP/Subnetting problems (IP allocation per network + routing tables).

![CiscoBuddy demo](Example.gif)

## Requirements

- **Go 1.21+** (for building) — tested on `go1.26.3`
- **git** (for the install one-liner)
- Linux, macOS, or Windows

Check your version with:

```bash
go version
```

If you don't have Go, install it with one of the one-liners below.

### Quick Go install — Linux (Debian/Ubuntu/Kali)

```bash
sudo apt-get update && sudo apt-get install -y golang-go git
```

### Quick Go install — Linux (official tarball, any distro)

```bash
curl -fsSLO https://go.dev/dl/go1.23.4.linux-amd64.tar.gz && sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz && echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && export PATH=$PATH:/usr/local/go/bin
```

### Quick Go install — Windows (PowerShell, winget)

```powershell
winget install --id GoLang.Go -e --source winget; winget install --id Git.Git -e --source winget
```

Otherwise, grab Go from <https://go.dev/dl/>.

## Install CiscoBuddy

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/papura-octavian/CiscoBuddy/main/install.sh | bash
```

### Windows (PowerShell)

```powershell
iwr -useb https://raw.githubusercontent.com/papura-octavian/CiscoBuddy/main/install.ps1 | iex
```

The command clones the repo into a temporary directory, builds the binary, and drops it somewhere on your `PATH` (see below). After that, open a new terminal and use `ciscobuddy ...`.

> On Windows, if PowerShell complains about execution policy, run this once:
> ```powershell
> Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned
> ```

## Usage

### Command syntax

```
ciscobuddy -ip <ip_start> <ip_end> -r <network_count> \
           -name <network_name_1> -dev <device_count_1> \
           -name <network_name_2> -dev <device_count_2> \
           ...
```

### Arguments

| Flag | Description |
|------|-------------|
| `-ip <start> <end>` | Available IP range (both inclusive) |
| `-r <N>` | Number of LAN networks |
| `-name <text>` | Network name (use quotes if it contains spaces) |
| `-dev <N>` | Device count, **router included** |

`-name` and `-dev` are repeated for each network, in order.

## Uninstall

```bash
# Linux / macOS
rm ~/.local/bin/ciscobuddy

# Windows
del %USERPROFILE%\bin\ciscobuddy.exe
```

## How it works 

1. **Per-LAN size:** `size = nextPow2(devices + 2)` (the +2 covers AR and AB).
2. **Router network size:** `size = nextPow2(N + 2)` where N = number of routers = number of networks.
3. **VLSM allocation:**
   - Sort all subnets by size, descending.
   - For each one, find the first free interval where it can be placed **aligned** (start address divisible by the subnet size).
   - Split the free interval and continue.
4. **Router interface assignment:** the router for network `i` gets `routerBase + 1 + i`.
5. **Routing table:** for the router of network `i`, one entry per every other network `j`, with next-hop = router `j`'s interface in the router network.

## Common errors

| Message | Cause |
|---------|-------|
| `insufficient address space for subnet of size X` | The IP range is too small to fit all subnets |
| `-r says N networks but got M -name/-dev pairs` | The number declared in `-r` doesn't match the number of `-name/-dev` pairs |
| `invalid IPv4 in -ip range` | One of the `-ip` addresses isn't a valid IPv4 |
| `end IP < start IP` | The range is reversed |

All errors are printed to `stderr` and the program exits with code `1`.

## Limitations

- IPv4 only.
- The router topology is always single-backbone (all routers on one shared network). Topologies with /30 point-to-point links between pairs or chains aren't supported yet.
- Network names with spaces must be quoted.

## License

Released under the [MIT License](LICENSE).

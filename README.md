# Scion

> Passive subdomain and domain enumeration. Fast, focused, pipe-friendly.

Scion is a passive reconnaissance tool for discovering subdomains and related domains associated with a target. It aggregates results across multiple public certificate transparency logs, DNS datasets, and optional API-backed sources — then deduplicates, optionally validates, and outputs in your format of choice.

Scion is based on [assetfinder](https://github.com/tomnomnom/assetfinder) by Tom Hudson (tomnomnom). See [CREDITS](./CREDITS).

---

## Performance

In testing against real bug bounty targets with no API keys configured, Scion consistently outperforms comparable tools:

| Tool | Results (paylution.com) | Keys Required |
|------|------------------------|---------------|
| subfinder | 69 | none |
| assetfinder | 83 | embedded |
| **Scion** | **264** | **none** |

---

## Install

**From source (requires Go 1.21+):**
```bash
go install github.com/RowanDark/scion@latest
```

**From release binary:**

Download the appropriate binary for your platform from the [Releases](https://github.com/RowanDark/scion/releases) page and place it in your `$PATH`.

**Add Go bin to PATH if needed:**
```bash
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc && source ~/.bashrc
```

---

## Usage

```
scion [flags] <domain>
```

### Basic Examples

```bash
# Enumerate all related domains and subdomains
scion example.com

# Subdomains only
scion --subs-only example.com

# JSON output written to file
scion --output json --out-file results.json example.com

# Only use specific sources
scion --sources crtsh,securitytrails example.com

# Validate which results actually resolve
scion --verify example.com

# Diff against a previous run to find new assets
scion --compare last_run.txt example.com

# Filter results against a scope file
scion --scope-file scope.txt example.com
```

### Pipeline Examples

```bash
# Feed directly into httpx
scion --subs-only example.com | httpx -silent

# Chain into nuclei
scion --subs-only --verify example.com | nuclei -t exposures/

# Output CSV for import into a spreadsheet or tracker
scion --output csv --out-file recon.csv example.com

# Pipe from a domain list
cat domains.txt | scion --subs-only
```

---

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--subs-only` | false | Return only subdomains of the target domain |
| `--output`, `-o` | `text` | Output format: `text`, `json`, `csv`, `md` |
| `--out-file`, `-f` | — | Write output to a file |
| `--timeout` | `30` | Per-source timeout in seconds. Note: some sources (e.g. Wayback Machine) have longer internal timeouts and will not be affected by `--timeout` values lower than their source default. |
| `--concurrency` | `5` | Max concurrent source queries |
| `--dns-concurrency` | `30` | Max concurrent DNS validation goroutines (used with `--verify`) |
| `--silent` | false | Suppress banner, warnings, and status output — domain results only |
| `--no-color` | false | Disable color output (auto-disabled when not a terminal) |
| `--verify` | false | DNS-validate results and annotate which resolve |
| `--scope-file` | — | Path to a file of in-scope domains; filter output to matches only |
| `--compare` | — | Path to a previous output file; highlight new findings |
| `--sources` | all | Comma-separated list of sources to query |
| `--list-sources` | — | Print all available sources with status and exit |
| `--version` | — | Print version and exit |

---

## Sources

### Free (no key required)

| Source | ID | Notes |
|--------|----|-------|
| crt.sh | `crtsh` | Certificate transparency logs |
| Certspotter | `certspotter` | Certificate transparency (Sectigo) |
| HackerTarget | `hackertarget` | Passive DNS |
| Wayback Machine | `wayback` | CDX API subdomain extraction |
| RapidDNS | `rapiddns` | Passive DNS dataset |
| AlienVault OTX | `alienvault` | Open threat exchange passive DNS — coverage varies by target |
| DNSRepo | `dnsrepo` | Public passive DNS dataset — coverage varies by target |
| DNSDumpster | `dnsdumpster` | Passive DNS and crawl data |
| Robtex | `robtex` | Passive DNS historical records |
| Anubis | `anubis` | Lightweight passive DNS |

### API-Backed (optional)

Set the relevant environment variable to enable. Sources are silently skipped if the key is not present (use `--list-sources` to check status).

| Source | ID | Environment Variable(s) | Registration |
|--------|----|--------------------------|-------------|
| VirusTotal | `virustotal` | `VT_API_KEY` | [virustotal.com](https://www.virustotal.com) |
| SecurityTrails | `securitytrails` | `ST_API_KEY` | [securitytrails.com](https://securitytrails.com) |
| Shodan | `shodan` | `SHODAN_API_KEY` | [shodan.io](https://shodan.io) |
| LeakIX | `leakix` | `LEAKIX_API_KEY` | [leakix.net](https://leakix.net) |
| Censys | `censys` | `CENSYS_API_ID` + `CENSYS_API_SECRET` | [search.censys.io/register](https://search.censys.io/register) |
| GitHub | `github` | `GITHUB_TOKEN` | [github.com/settings/tokens](https://github.com/settings/tokens) |
| BufferOver | `bufferover` | `BUFFEROVER_KEY` | [tls.bufferover.run](https://tls.bufferover.run) |
| FullHunt | `fullhunt` | `FULLHUNT_KEY` | [fullhunt.io](https://fullhunt.io) |
| URLScan.io | `urlscan` | `URLSCAN_API_KEY` | [urlscan.io](https://urlscan.io) |
| Netlas | `netlas` | `NETLAS_API_KEY` | [netlas.io](https://netlas.io) |
| RedHuntLabs | `redhuntlabs` | `REDHUNTLABS_API_KEY` | [reconapi.redhuntlabs.com](https://reconapi.redhuntlabs.com) |
| BeVigil | `bevigil` | `BEVIGIL_API_KEY` | [bevigil.com/osint-api](https://bevigil.com/osint-api) |

---

## Managing API Keys

Export keys in your shell profile (`~/.bashrc`, `~/.zshrc`):

```bash
export VT_API_KEY="your_key_here"
export ST_API_KEY="your_key_here"
export SHODAN_API_KEY="your_key_here"
export LEAKIX_API_KEY="your_key_here"
export CENSYS_API_ID="your_api_id"
export CENSYS_API_SECRET="your_api_secret"
export GITHUB_TOKEN="your_token_here"
export BUFFEROVER_KEY="your_key_here"
export FULLHUNT_KEY="your_key_here"
export URLSCAN_API_KEY="your_key_here"
export NETLAS_API_KEY="your_key_here"
export REDHUNTLABS_API_KEY="your_key_here"
export BEVIGIL_API_KEY="your_key_here"
```

### URLScan.io Setup

URLScan.io has a free tier with 1,000 searches per day — no payment required.

1. Go to https://urlscan.io and create a free account
2. Go to Settings → API Keys
3. Click New Key, give it a name (e.g. `scion`)
4. Copy the key and export it:

```bash
export URLSCAN_API_KEY="your_key_here"
```

### GitHub Token Setup

The GitHub source only requires read access to public repositories. A fine-grained personal access token is recommended:

1. Go to [github.com/settings/tokens](https://github.com/settings/tokens)
2. Click **Generate new token (classic)**
3. Select scope: `public_repo` only — no write permissions needed
4. Copy the token and export it:

```bash
export GITHUB_TOKEN="your_token_here"
```

### Netlas Setup

Free tier gives 50 API requests/day — sufficient for regular recon use.

1. Go to https://netlas.io and create a free account
2. Go to your profile page and copy your API key
3. Export: `export NETLAS_API_KEY="your_key_here"`

### RedHuntLabs Setup

Free community tier available.

1. Go to https://reconapi.redhuntlabs.com
2. Register for a community account
3. Copy your API key from the dashboard
4. Export: `export REDHUNTLABS_API_KEY="your_key_here"`

### BeVigil Setup

Free API key available — no payment required.

1. Go to https://bevigil.com/osint-api
2. Register for an account
3. Copy your API key
4. Export: `export BEVIGIL_API_KEY="your_key_here"`

---

## Output Formats

### text (default)
One domain per line. Ideal for piping into other tools.
```
sub1.example.com
sub2.example.com
mail.example.com
```

### json
Structured output with source attribution and metadata.
```json
{
  "target": "example.com",
  "timestamp": "2026-05-03T12:00:00Z",
  "total": 3,
  "sources_used": ["crtsh", "certspotter", "rapiddns"],
  "wildcard_detected": false,
  "results": [
    { "domain": "sub1.example.com", "source": "crtsh", "resolves": true },
    { "domain": "sub2.example.com", "source": "securitytrails", "resolves": true },
    { "domain": "mail.example.com", "source": "wayback", "resolves": false }
  ]
}
```

### csv
```
domain,source,resolves,new,wildcard
sub1.example.com,crtsh,true,false,false
sub2.example.com,securitytrails,true,false,false
mail.example.com,wayback,false,false,false
```

### md
```markdown
| Domain | Source | Resolves | New |
|--------|--------|----------|-----|
| sub1.example.com | crtsh | ✓ | — |
| sub2.example.com | securitytrails | ✓ | — |
| mail.example.com | wayback | ✗ | — |
```

---

## Features

### DNS Validation (`--verify`)
After collecting results from all sources, Scion performs a lightweight A/CNAME lookup on each discovered domain to determine if it actively resolves. Unresolvable domains are included in output but flagged — useful for filtering ghost subdomains before passing results to downstream tools.

Wildcard DNS is auto-detected at startup. If `*.target.com` resolves, Scion will warn you and annotate results accordingly, since wildcard responses pollute validation results.

Use `--dns-concurrency` to control how many DNS lookups run in parallel (default: 30).

### Diff Mode (`--compare`)
Point `--compare` at a previous Scion output file (any format) and Scion will highlight domains that are new since that run. Useful for monitoring a target across multiple bug bounty sessions or tracking infrastructure changes over time.

```bash
# First run — save baseline
scion --output text --out-file baseline.txt example.com

# Later run — show only new results
scion --compare baseline.txt example.com
```

New domains are prefixed with `[NEW]` in text mode, or tagged with `"new": true` in JSON.

### Scope Filtering (`--scope-file`)
Provide a newline-delimited file of in-scope domains or wildcard patterns. Scion will filter output to only matching results — no grep chaining required.

```
# scope.txt
*.example.com
admin.example.net
*.qa*.example.com
```

```bash
scion --scope-file scope.txt example.com
```

Supports exact matches and glob-style wildcard patterns including mid-string wildcards. Lines beginning with `#` are treated as comments.

### Source Selection (`--sources`, `--list-sources`)
Run `--list-sources` to see all sources, their IDs, and whether required API keys are present:

```
Source           ID               Key Required          Status
──────────────────────────────────────────────────────────────
crt.sh           crtsh            No                    ✓ ready
Certspotter      certspotter      No                    ✓ ready
HackerTarget     hackertarget     No                    ✓ ready
Wayback Machine  wayback          No                    ✓ ready
RapidDNS         rapiddns         No                    ✓ ready
AlienVault OTX   alienvault       No                    ✓ ready
DNSRepo          dnsrepo          No                    ✓ ready
DNSDumpster      dnsdumpster      No                    ✓ ready
Robtex           robtex           No                    ✓ ready
Anubis           anubis           No                    ✓ ready
VirusTotal       virustotal       VT_API_KEY            ✗ key not set
SecurityTrails   securitytrails   ST_API_KEY            ✗ key not set
Shodan           shodan           SHODAN_API_KEY        ✗ key not set
LeakIX           leakix           LEAKIX_API_KEY        ✗ key not set
Censys           censys           CENSYS_API_ID/SECRET  ✗ key not set
GitHub           github           GITHUB_TOKEN          ✗ key not set
BufferOver       bufferover       BUFFEROVER_KEY        ✗ key not set
FullHunt         fullhunt         FULLHUNT_KEY          ✗ key not set
URLScan.io       urlscan          URLSCAN_API_KEY       ✗ key not set
Netlas           netlas           NETLAS_API_KEY        ✗ key not set
RedHuntLabs      redhuntlabs      REDHUNTLABS_API_KEY   ✗ key not set
BeVigil          bevigil          BEVIGIL_API_KEY       ✗ key not set
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success — results found |
| `1` | Error — source failure, bad flags, etc. |
| `2` | No results found |

---

## Contributing

Pull requests are welcome, especially for new passive sources. To add a source, implement the `Source` interface in `sources/` and register it in `main.go`. Each source should handle its own timeout context and return a deduplicated `[]string` of domains.

---

## Legal

Scion performs **passive reconnaissance only**. It does not interact directly with target infrastructure. Always ensure you have authorization before conducting any reconnaissance activity.

---

## Credits

Scion is based on [assetfinder](https://github.com/tomnomnom/assetfinder) by Tom Hudson.
Original copyright (c) 2019 Tom Hudson — MIT License.
See [CREDITS](./CREDITS) for full attribution.

---

## License

MIT — see [LICENSE](./LICENSE)

# Scion

Passive subdomain and domain enumeration. Fast, focused, pipe-friendly.

Scion is a passive reconnaissance tool for discovering subdomains and related domains associated with a target. It aggregates results across multiple public certificate transparency logs, DNS datasets, and optional API-backed sources — then deduplicates, optionally validates, and outputs in your format of choice.
Scion is based on assetfinder by Tom Hudson (tomnomnom). See CREDITS.

Install
From source (requires Go 1.21+):
bashgo install github.com/RowanDark/scion@latest
From release binary:
Download the appropriate binary for your platform from the Releases page and place it in your $PATH.

Usage
scion [flags] <domain>
Basic Examples
bash# Enumerate all related domains and subdomains
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
Pipeline Examples
bash# Feed directly into httpx
scion --subs-only example.com | httpx -silent

# Chain into nuclei
scion --subs-only --verify example.com | nuclei -t exposures/

# Output CSV for import into a spreadsheet or tracker
scion --output csv --out-file recon.csv example.com

Flags
FlagDefaultDescription--subs-onlyfalseReturn only subdomains of the target domain--output, -otextOutput format: text, json, csv, md--out-file, -f—Write output to a file--timeout30Per-source timeout in seconds--concurrency5Max concurrent source queries--silentfalseSuppress banner, warnings, and status output--verifyfalseDNS-validate results and annotate which resolve--scope-file—Path to a file of in-scope domains; filter output to matches only--compare—Path to a previous output file; highlight new findings--sourcesallComma-separated list of sources to query--list-sources—Print all available sources with status and exit

Sources
Free (no key required)
SourceIDNotescrt.shcrtshCertificate transparency logsCertspottercertspotterCertificate transparency (Sectigo)HackerTargethackertargetPassive DNSWayback MachinewaybackCDX API subdomain extractionRapidDNSrapiddnsPassive DNS datasetAlienVault OTXalienvaultOpen threat exchange passive DNSLeakIXleakixPublic endpoint, exposed asset indexing
API-Backed (optional)
Set the relevant environment variable to enable. Sources are silently skipped if the key is not present (use --list-sources to check status).
SourceIDEnvironment Variable(s)VirusTotalvirustotalVT_API_KEYSecurityTrailssecuritytrailsST_API_KEYShodan InternetDBshodanSHODAN_API_KEYFacebook CTfacebookFB_APP_ID + FB_APP_SECRET
Managing API Keys
Export keys in your shell profile (~/.bashrc, ~/.zshrc):
bashexport VT_API_KEY="your_key_here"
export ST_API_KEY="your_key_here"
export SHODAN_API_KEY="your_key_here"
export FB_APP_ID="your_app_id"
export FB_APP_SECRET="your_app_secret"

Output Formats
text (default)
One domain per line. Ideal for piping into other tools.
sub1.example.com
sub2.example.com
mail.example.com
json
Structured output with source attribution and metadata.
json{
  "target": "example.com",
  "timestamp": "2026-05-03T12:00:00Z",
  "total": 3,
  "results": [
    { "domain": "sub1.example.com", "source": "crtsh", "resolves": true },
    { "domain": "sub2.example.com", "source": "securitytrails", "resolves": true },
    { "domain": "mail.example.com", "source": "wayback", "resolves": false }
  ]
}
csv
domain,source,resolves
sub1.example.com,crtsh,true
sub2.example.com,securitytrails,true
mail.example.com,wayback,false
md
markdown| Domain | Source | Resolves |
|--------|--------|----------|
| sub1.example.com | crtsh | ✓ |
| sub2.example.com | securitytrails | ✓ |
| mail.example.com | wayback | ✗ |

Features
DNS Validation (--verify)
After collecting results from all sources, Scion performs a lightweight A/CNAME lookup on each discovered domain to determine if it actively resolves. Unresolvable domains are included in output but flagged — useful for filtering ghost subdomains before passing results to downstream tools.
Wildcard DNS is auto-detected at startup. If *.target.com resolves, Scion will warn you and annotate results accordingly, since wildcard responses pollute validation results.
Diff Mode (--compare)
Point --compare at a previous Scion output file (any format) and Scion will highlight domains that are new since that run. Useful for monitoring a target across multiple bug bounty sessions or tracking infrastructure changes over time.
bash# First run — save baseline
scion --output text --out-file baseline.txt example.com

# Later run — show only new results
scion --compare baseline.txt example.com
New domains are prefixed with [NEW] in text mode, or tagged with "new": true in JSON.
Scope Filtering (--scope-file)
Provide a newline-delimited file of in-scope domains or wildcard patterns. Scion will filter output to only matching results — no grep chaining required.
# scope.txt
*.example.com
admin.example.net
*.internal.example.org
bashscion --scope-file scope.txt example.com
Source Selection (--sources, --list-sources)
Run --list-sources to see all sources, their IDs, and whether required API keys are present:
Source           ID               Status
──────────────────────────────────────────
crt.sh           crtsh            ✓ ready
Certspotter      certspotter      ✓ ready
HackerTarget     hackertarget     ✓ ready
Wayback Machine  wayback          ✓ ready
RapidDNS         rapiddns         ✓ ready
AlienVault OTX   alienvault       ✓ ready
LeakIX           leakix           ✓ ready
VirusTotal       virustotal       ✗ missing VT_API_KEY
SecurityTrails   securitytrails   ✗ missing ST_API_KEY
Shodan           shodan           ✗ missing SHODAN_API_KEY
Facebook CT      facebook         ✗ missing FB_APP_ID / FB_APP_SECRET

Exit Codes
CodeMeaning0Success — results found1Error — source failure, bad flags, etc.2No results found

Contributing
Pull requests are welcome, especially for new passive sources. To add a source, implement the Source interface in sources/ and register it in main.go. Each source should handle its own timeout context and return a deduplicated []string of domains.

Legal
Scion performs passive reconnaissance only. It does not interact directly with target infrastructure. Always ensure you have authorization before conducting any reconnaissance activity.

Credits
Scion is based on assetfinder by Tom Hudson.
Original copyright (c) 2019 Tom Hudson — MIT License.
See CREDITS for full attribution.

License
MIT — see LICENSE

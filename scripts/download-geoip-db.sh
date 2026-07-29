#!/bin/bash
# Download MaxMind GeoLite2-City database for GeoIP access conditions.
#
# Requires a MaxMind license key (free account at https://www.maxmind.com/en/geolite2/signup).
# Set MAXMIND_LICENSE_KEY environment variable before running.
#
# Usage:
#   MAXMIND_LICENSE_KEY=your_key ./scripts/download-geoip-db.sh
#
# The database is saved to data/GeoLite2-City.mmdb (gitignored).

set -euo pipefail

if [ -z "${MAXMIND_LICENSE_KEY:-}" ]; then
  echo "ERROR: MAXMIND_LICENSE_KEY environment variable is required."
  echo ""
  echo "Get a free license key at: https://www.maxmind.com/en/geolite2/signup"
  echo "Then run: MAXMIND_LICENSE_KEY=your_key MAXMIND_ACCOUNT_ID=your_id $0"
  exit 1
fi

if [ -z "${MAXMIND_ACCOUNT_ID:-}" ]; then
  echo "ERROR: MAXMIND_ACCOUNT_ID environment variable is required."
  echo ""
  echo "Find your account ID at: https://www.maxmind.com/en/accounts/current"
  echo "Then run: MAXMIND_LICENSE_KEY=your_key MAXMIND_ACCOUNT_ID=your_id $0"
  exit 1
fi

OUTDIR="$(cd "$(dirname "$0")/.." && pwd)/data"
mkdir -p "$OUTDIR"

TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE" "${TMPFILE}.tar.gz"' EXIT

echo "Downloading GeoLite2-City database..."
curl -sSfL -o "${TMPFILE}.tar.gz" \
  -u "${MAXMIND_ACCOUNT_ID}:${MAXMIND_LICENSE_KEY}" \
  "https://download.maxmind.com/geoip/databases/GeoLite2-City/download?suffix=tar.gz"

echo "Extracting..."
TMPDIR_EXTRACT=$(mktemp -d)
tar -xzf "${TMPFILE}.tar.gz" -C "$TMPDIR_EXTRACT"
find "$TMPDIR_EXTRACT" -name 'GeoLite2-City.mmdb' -exec mv {} "$OUTDIR/GeoLite2-City.mmdb" \;
rm -rf "$TMPDIR_EXTRACT"

if [ -f "$OUTDIR/GeoLite2-City.mmdb" ]; then
  echo "Done: $OUTDIR/GeoLite2-City.mmdb"
else
  echo "ERROR: Extraction failed — GeoLite2-City.mmdb not found."
  exit 1
fi

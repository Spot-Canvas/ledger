#!/usr/bin/env bash
#
# One-time setup: reserve a static external IP for the trader Cloud Run service
# and wire it through a Cloud NAT so all outbound traffic (e.g. Binance API)
# exits from a predictable IP.
#
# Architecture:
#   Cloud Run (Direct VPC Egress) → default VPC → Cloud NAT → static IP → Internet
#
# Usage:
#   ./scripts/setup-static-ip.sh              # Apply against signalngn-prod
#   ./scripts/setup-static-ip.sh --dry-run    # Preview only
#
# Prerequisites:
#   - gcloud authenticated with signalngn-prod IAM permissions
#   - compute.googleapis.com enabled on the project
#

set -euo pipefail

PROJECT="signalngn-prod"
REGION="europe-west1"
NETWORK="default"
SUBNET="default"
ADDRESS_NAME="trader-nat-ip"
ROUTER_NAME="trader-router"
NAT_NAME="trader-nat"
DRY_RUN=false

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_step()  { echo -e "${BLUE}[STEP]${NC}  $1"; }
log_value() { echo -e "${CYAN}        $1${NC}"; }

while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run) DRY_RUN=true; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

run_cmd() {
    if [[ "$DRY_RUN" == true ]]; then
        echo "  \$ $*"
    else
        "$@"
    fi
}

exists_cmd() {
    # Run silently; return the exit code.
    "$@" &>/dev/null
}

echo ""
log_info "======================================"
log_info "Trader — Static Egress IP Setup"
log_info "======================================"
echo ""
log_info "Project : $PROJECT"
log_info "Region  : $REGION"
log_info "Network : $NETWORK"
echo ""
[[ "$DRY_RUN" == true ]] && log_warn "DRY RUN — nothing will be created" && echo ""

# ── Step 1: Reserve static external IP ───────────────────────
log_step "1/3  Reserve static external IP address ($ADDRESS_NAME)..."
if exists_cmd gcloud compute addresses describe "$ADDRESS_NAME" \
        --region="$REGION" --project="$PROJECT"; then
    log_warn "Address '$ADDRESS_NAME' already exists — skipping."
else
    run_cmd gcloud compute addresses create "$ADDRESS_NAME" \
        --region="$REGION" \
        --project="$PROJECT"
fi

# ── Step 2: Cloud Router ──────────────────────────────────────
log_step "2/3  Create Cloud Router ($ROUTER_NAME)..."
if exists_cmd gcloud compute routers describe "$ROUTER_NAME" \
        --region="$REGION" --project="$PROJECT"; then
    log_warn "Router '$ROUTER_NAME' already exists — skipping."
else
    run_cmd gcloud compute routers create "$ROUTER_NAME" \
        --region="$REGION" \
        --network="$NETWORK" \
        --project="$PROJECT"
fi

# ── Step 3: Cloud NAT with the static IP ─────────────────────
log_step "3/3  Create Cloud NAT ($NAT_NAME) pinned to static IP..."
if exists_cmd gcloud compute routers nats describe "$NAT_NAME" \
        --router="$ROUTER_NAME" --region="$REGION" --project="$PROJECT"; then
    log_warn "NAT '$NAT_NAME' already exists — skipping."
else
    run_cmd gcloud compute routers nats create "$NAT_NAME" \
        --router="$ROUTER_NAME" \
        --region="$REGION" \
        --nat-external-ip-pool="$ADDRESS_NAME" \
        --nat-all-subnet-ip-ranges \
        --project="$PROJECT"
fi

# ── Print the IP ──────────────────────────────────────────────
echo ""
if [[ "$DRY_RUN" != true ]]; then
    STATIC_IP=$(gcloud compute addresses describe "$ADDRESS_NAME" \
        --region="$REGION" \
        --project="$PROJECT" \
        --format="value(address)")

    log_info "======================================"
    log_info "SETUP COMPLETE"
    log_info "======================================"
    echo ""
    echo -e "  Static egress IP:  ${CYAN}${STATIC_IP}${NC}"
    echo ""
    echo "  ➜  Add this IP to your Binance API key's IP allowlist."
    echo "  ➜  Then redeploy the Cloud Run service so Direct VPC Egress"
    echo "     is active (cloudbuild-prod.yaml already includes the flags)."
else
    log_info "Dry run complete."
    echo ""
    echo "  Next step: run without --dry-run, then add the printed IP"
    echo "  to your Binance API key's IP allowlist."
fi
echo ""

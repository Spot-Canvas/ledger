-- Migration 005: Rebuild positions again with improved margin scale factor.
--
-- Migration 004 fixed futures P&L to use leverage when present, but fell back
-- to notional (1x) when leverage was missing. This migration adds support for
-- deriving the scale factor from the margin field (margin / cost_basis) so that
-- trades sent without explicit leverage but with a margin value are also
-- correctly scaled to account-impact P&L.
--
-- Truncate positions so the application rebuilds them from trades on startup.

TRUNCATE TABLE ledger_positions;

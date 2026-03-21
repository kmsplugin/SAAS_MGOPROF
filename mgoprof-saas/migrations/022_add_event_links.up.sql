-- Migration 022: per-event streaming/presentation links separated by role.
-- speaker_link  → URL for presenters / moderators with elevated permissions
-- viewer_link   → URL for regular attendees (view-only)
-- cabinet_link retains the original general link for backwards compatibility.

ALTER TABLE reg_events
    ADD COLUMN IF NOT EXISTS speaker_link TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS viewer_link  TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN reg_events.speaker_link IS
    'URL for speakers/moderators: LiveKit room with publisher role, Zoom host link, etc.';
COMMENT ON COLUMN reg_events.viewer_link IS
    'URL for attendees: viewer-only stream link; falls back to cabinet_link if empty.';

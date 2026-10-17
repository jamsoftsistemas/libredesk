package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_20_0 adds the "transcription" AI provider type used to transcribe audio attachments.
func V2_20_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		ALTER TABLE ai_providers DROP CONSTRAINT IF EXISTS constraint_ai_providers_on_type;
		ALTER TABLE ai_providers ADD CONSTRAINT constraint_ai_providers_on_type CHECK (type IN ('completion', 'embedding', 'transcription'));
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		INSERT INTO ai_providers ("name", provider, type, config, is_default)
		VALUES ('transcription', 'openai', 'transcription', '{"api_key": "", "base_url": "https://openrouter.ai/api/v1", "model": "google/gemini-2.5-flash"}'::jsonb, false)
		ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}
	return nil
}

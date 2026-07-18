CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS writing_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    slug VARCHAR(100) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT DEFAULT '',
    tone VARCHAR(100) DEFAULT '',
    perspective VARCHAR(100) DEFAULT '',
    audience VARCHAR(200) DEFAULT '',
    expertise_level VARCHAR(50) DEFAULT 'general',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    vocabulary_tags TEXT[] DEFAULT '{}',
    allowed_connectors TEXT[] DEFAULT '{}',
    preferred_sentence_length VARCHAR(50) DEFAULT 'medium',
    paragraph_size_min INT DEFAULT 3,
    paragraph_size_max INT DEFAULT 8,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(site_id, slug)
);

CREATE INDEX idx_wp_site ON writing_profiles(site_id);
CREATE INDEX idx_wp_slug ON writing_profiles(slug);
CREATE INDEX idx_wp_language ON writing_profiles(language);

CREATE TABLE IF NOT EXISTS writing_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    profile_id UUID REFERENCES writing_profiles(id) ON DELETE CASCADE,
    rule_key VARCHAR(100) NOT NULL,
    category VARCHAR(100) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    priority INT NOT NULL DEFAULT 5,
    config JSONB DEFAULT '{}'::jsonb,
    description TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wr_site ON writing_rules(site_id);
CREATE INDEX idx_wr_profile ON writing_rules(profile_id);
CREATE INDEX idx_wr_rule_key ON writing_rules(rule_key);
CREATE INDEX idx_wr_category ON writing_rules(category);

CREATE TABLE IF NOT EXISTS writing_personas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    profile_id UUID REFERENCES writing_profiles(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    title VARCHAR(200) DEFAULT '',
    bio TEXT DEFAULT '',
    voice_traits TEXT[] DEFAULT '{}',
    vocabulary_style TEXT[] DEFAULT '{}',
    sentence_patterns TEXT[] DEFAULT '{}',
    expertise_areas TEXT[] DEFAULT '{}',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wp2_site ON writing_personas(site_id);
CREATE INDEX idx_wp2_profile ON writing_personas(profile_id);
CREATE INDEX idx_wp2_language ON writing_personas(language);

CREATE TABLE IF NOT EXISTS vocabulary_sets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    category VARCHAR(100) DEFAULT 'general',
    words TEXT[] NOT NULL DEFAULT '{}',
    replacements TEXT[][] DEFAULT '{}',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    tags TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vs_site ON vocabulary_sets(site_id);
CREATE INDEX idx_vs_category ON vocabulary_sets(category);
CREATE INDEX idx_vs_language ON vocabulary_sets(language);

CREATE TABLE IF NOT EXISTS transition_library (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    category VARCHAR(100) NOT NULL,
    phrase TEXT NOT NULL,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    formality VARCHAR(50) DEFAULT 'neutral',
    usage_count INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tl_site ON transition_library(site_id);
CREATE INDEX idx_tl_category ON transition_library(category);
CREATE INDEX idx_tl_language ON transition_library(language);
CREATE INDEX idx_tl_formality ON transition_library(formality);

CREATE TABLE IF NOT EXISTS style_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    profile_id UUID REFERENCES writing_profiles(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    pattern_type VARCHAR(100) NOT NULL,
    pattern TEXT NOT NULL,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    tags TEXT[] DEFAULT '{}',
    effectiveness_score NUMERIC(3,2) DEFAULT 0,
    usage_count INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sp_site ON style_patterns(site_id);
CREATE INDEX idx_sp_profile ON style_patterns(profile_id);
CREATE INDEX idx_sp_type ON style_patterns(pattern_type);
CREATE INDEX idx_sp_language ON style_patterns(language);

CREATE TABLE IF NOT EXISTS sentence_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    profile_id UUID REFERENCES writing_profiles(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    template TEXT NOT NULL,
    category VARCHAR(100) DEFAULT 'general',
    variables TEXT[] DEFAULT '{}',
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    formality VARCHAR(50) DEFAULT 'neutral',
    usage_count INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_st_site ON sentence_templates(site_id);
CREATE INDEX idx_st_profile ON sentence_templates(profile_id);
CREATE INDEX idx_st_category ON sentence_templates(category);
CREATE INDEX idx_st_language ON sentence_templates(language);

CREATE TABLE IF NOT EXISTS humanization_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    profile_id UUID REFERENCES writing_profiles(id) ON DELETE CASCADE,
    source_text TEXT NOT NULL,
    humanized_text TEXT NOT NULL,
    burstiness_score NUMERIC(5,2) DEFAULT 0,
    perplexity_score NUMERIC(5,2) DEFAULT 0,
    repetition_score NUMERIC(5,2) DEFAULT 0,
    passive_voice_score NUMERIC(5,2) DEFAULT 0,
    rhythm_score NUMERIC(5,2) DEFAULT 0,
    flow_score NUMERIC(5,2) DEFAULT 0,
    rules_applied TEXT[] DEFAULT '{}',
    transformations JSONB DEFAULT '[]'::jsonb,
    language VARCHAR(10) NOT NULL DEFAULT 'pt',
    word_count_original INT DEFAULT 0,
    word_count_humanized INT DEFAULT 0,
    duration_ms INT DEFAULT 0,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_hh_site ON humanization_history(site_id);
CREATE INDEX idx_hh_profile ON humanization_history(profile_id);
CREATE INDEX idx_hh_language ON humanization_history(language);
CREATE INDEX idx_hh_created ON humanization_history(created_at);

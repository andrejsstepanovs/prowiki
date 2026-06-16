-- feature_interactions
CREATE INDEX IF NOT EXISTS idx_fi_from_to
    ON feature_interactions(from_feature_id ASC, to_feature_id ASC);

-- style_anomalies
CREATE INDEX IF NOT EXISTS idx_sa_file_version
    ON style_anomalies(file_version_id ASC);
CREATE INDEX IF NOT EXISTS idx_sa_code_style
    ON style_anomalies(code_style_id ASC);

-- entities
CREATE INDEX IF NOT EXISTS idx_entities_project
    ON entities(project_id ASC);
CREATE UNIQUE INDEX IF NOT EXISTS uq_entities_project_name_type
    ON entities(project_id, name, type);

-- features
CREATE UNIQUE INDEX IF NOT EXISTS uq_features_project_name
    ON features(project_id, name);

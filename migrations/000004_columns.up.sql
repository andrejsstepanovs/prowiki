-- Add payload column to job_queue for storing secondary context (e.g. second feature ID for intersection synthesis)
ALTER TABLE job_queue ADD COLUMN payload TEXT NOT NULL DEFAULT '';

-- Add fs_location column to projects for storing the filesystem path used by prowiki init
ALTER TABLE projects ADD COLUMN fs_location TEXT NOT NULL DEFAULT '';

-- Seed missing prompt stages that were not included in 000002_seed_prompts.up.sql
INSERT OR IGNORE INTO prompt_registry (stage, template, version, is_active) VALUES (
    'level_4_edge_case',
    'Analyze the following code for edge cases and unusual conditions:

{{.Content}}

Provide a list of edge case annotations, each with a description field.',
    1,
    1
);

INSERT OR IGNORE INTO prompt_registry (stage, template, version, is_active) VALUES (
    'intersection_synthesis',
    'Analyze the relationship between the following two features:

Feature A: {{.FeatureA}}
Feature B: {{.FeatureB}}

Describe how these features interact or relate to each other.',
    1,
    1
);

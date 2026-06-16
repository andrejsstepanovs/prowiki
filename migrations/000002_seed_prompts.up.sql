INSERT INTO prompt_registry (stage, template, version, is_active) VALUES 
('level_1_overview', 'You are an expert analyzing code. Summarize this file and list its core features.
File: {{.Content}}
Return JSON: {"summary": "...", "features": [{"name": "...", "description": "..."}]}', 1, 1),

('level_2_entity', 'You are an expert. Find all entities (structs, classes, funcs) in this file.
File: {{.Content}}
Return JSON: {"entities": [{"name": "...", "type": "...", "description": "..."}]}', 1, 1),

('level_3_feature', 'You are an expert analyzing architectural dependencies. How do these features interact?
Feature 1: {{.Feature1}}
Feature 2: {{.Feature2}}
Return JSON: {"interactions": [{"to_feature_id": 1, "description": "..."}]}', 1, 1),

('style_evaluation', 'Does this code follow the given style rule?
Rule: {{.Rule}}
Code: {{.Content}}
Return JSON: {"anomalies": [{"rationale": "..."}]}', 1, 1);

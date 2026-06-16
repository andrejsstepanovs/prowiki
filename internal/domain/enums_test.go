package domain

import (
	"testing"
)

func TestParseJobStatus(t *testing.T) {
	cases := []struct {
		input   string
		valid   bool
		status  JobStatus
	}{
		{"pending", true, JobStatusPending},
		{"processing", true, JobStatusProcessing},
		{"completed", true, JobStatusCompleted},
		{"failed", true, JobStatusFailed},
		{"dead_lettered", true, JobStatusDeadLettered},
		{"invalid", false, ""},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			status, err := ParseJobStatus(c.input)
			if c.valid {
				if err != nil {
					t.Errorf("expected no error for %s, got %v", c.input, err)
				}
				if status != c.status {
					t.Errorf("expected %s, got %s", c.status, status)
				}
				if status.String() != c.input {
					t.Errorf("expected string %s, got %s", c.input, status.String())
				}
			} else {
				if err == nil {
					t.Errorf("expected error for %s", c.input)
				}
			}
		})
	}
}

func TestParseEntityType(t *testing.T) {
	cases := []struct {
		input  string
		valid  bool
		entity EntityType
	}{
		{"file", true, EntityTypeFile},
		{"feature", true, EntityTypeFeature},
		{"entity", true, EntityTypeEntity},
		{"intersection", true, EntityTypeIntersection},
		{"macro_pipeline", true, EntityTypeMacroPipeline},
		{"style", true, EntityTypeStyle},
		{"invalid", false, ""},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			entity, err := ParseEntityType(c.input)
			if c.valid {
				if err != nil {
					t.Errorf("expected no error for %s, got %v", c.input, err)
				}
				if entity != c.entity {
					t.Errorf("expected %s, got %s", c.entity, entity)
				}
				if entity.String() != c.input {
					t.Errorf("expected string %s, got %s", c.input, entity.String())
				}
			} else {
				if err == nil {
					t.Errorf("expected error for %s", c.input)
				}
			}
		})
	}
}

func TestParseStage(t *testing.T) {
	cases := []struct {
		input string
		valid bool
		stage Stage
	}{
		{"level_1_overview", true, StageLevel1Overview},
		{"level_2_entity", true, StageLevel2Entity},
		{"level_3_feature", true, StageLevel3Feature},
		{"level_4_edge_case", true, StageLevel4EdgeCase},
		{"style_evaluation", true, StageStyleEvaluation},
		{"intersection_synthesis", true, StageIntersectionSynthesis},
		{"macro_synthesis", true, StageMacroSynthesis},
		{"invalid", false, ""},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			stage, err := ParseStage(c.input)
			if c.valid {
				if err != nil {
					t.Errorf("expected no error for %s, got %v", c.input, err)
				}
				if stage != c.stage {
					t.Errorf("expected %s, got %s", c.stage, stage)
				}
				if stage.String() != c.input {
					t.Errorf("expected string %s, got %s", c.input, stage.String())
				}
			} else {
				if err == nil {
					t.Errorf("expected error for %s", c.input)
				}
			}
		})
	}
}

func TestParseModelTier(t *testing.T) {
	cases := []struct {
		input string
		valid bool
		tier  ModelTier
	}{
		{"tier_1", true, ModelTier1},
		{"tier_2", true, ModelTier2},
		{"invalid", false, ""},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			tier, err := ParseModelTier(c.input)
			if c.valid {
				if err != nil {
					t.Errorf("expected no error for %s, got %v", c.input, err)
				}
				if tier != c.tier {
					t.Errorf("expected %s, got %s", c.tier, tier)
				}
				if tier.String() != c.input {
					t.Errorf("expected string %s, got %s", c.input, tier.String())
				}
			} else {
				if err == nil {
					t.Errorf("expected error for %s", c.input)
				}
			}
		})
	}
}

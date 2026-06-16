package domain

import "fmt"

type JobStatus string

const (
	JobStatusPending       JobStatus = "pending"
	JobStatusProcessing    JobStatus = "processing"
	JobStatusCompleted     JobStatus = "completed"
	JobStatusFailed        JobStatus = "failed"
	JobStatusDeadLettered  JobStatus = "dead_lettered"
)

func (s JobStatus) String() string { return string(s) }

func ParseJobStatus(s string) (JobStatus, error) {
	switch JobStatus(s) {
	case JobStatusPending, JobStatusProcessing, JobStatusCompleted, JobStatusFailed, JobStatusDeadLettered:
		return JobStatus(s), nil
	default:
		return "", fmt.Errorf("invalid JobStatus: %s", s)
	}
}

type EntityType string

const (
	EntityTypeFile           EntityType = "file"
	EntityTypeFeature        EntityType = "feature"
	EntityTypeEntity         EntityType = "entity"
	EntityTypeIntersection   EntityType = "intersection"
	EntityTypeMacroPipeline  EntityType = "macro_pipeline"
	EntityTypeStyle          EntityType = "style"
)

func (t EntityType) String() string { return string(t) }

func ParseEntityType(s string) (EntityType, error) {
	switch EntityType(s) {
	case EntityTypeFile, EntityTypeFeature, EntityTypeEntity, EntityTypeIntersection, EntityTypeMacroPipeline, EntityTypeStyle:
		return EntityType(s), nil
	default:
		return "", fmt.Errorf("invalid EntityType: %s", s)
	}
}

type Stage string

const (
	StageLevel1Overview        Stage = "level_1_overview"
	StageLevel2Entity          Stage = "level_2_entity"
	StageLevel3Feature         Stage = "level_3_feature"
	StageLevel4EdgeCase        Stage = "level_4_edge_case"
	StageStyleEvaluation       Stage = "style_evaluation"
	StageIntersectionSynthesis Stage = "intersection_synthesis"
	StageMacroSynthesis        Stage = "macro_synthesis"
)

func (s Stage) String() string { return string(s) }

func ParseStage(st string) (Stage, error) {
	switch Stage(st) {
	case StageLevel1Overview, StageLevel2Entity, StageLevel3Feature, StageLevel4EdgeCase, StageStyleEvaluation, StageIntersectionSynthesis, StageMacroSynthesis:
		return Stage(st), nil
	default:
		return "", fmt.Errorf("invalid Stage: %s", st)
	}
}

type ModelTier string

const (
	ModelTier1 ModelTier = "tier_1"
	ModelTier2 ModelTier = "tier_2"
)

func (t ModelTier) String() string { return string(t) }

func ParseModelTier(s string) (ModelTier, error) {
	switch ModelTier(s) {
	case ModelTier1, ModelTier2:
		return ModelTier(s), nil
	default:
		return "", fmt.Errorf("invalid ModelTier: %s", s)
	}
}

package extract

// OverviewSchema represents the expected JSON structure from StageLevel1Overview
type OverviewSchema struct {
	Summary  string          `json:"summary"`
	Features []FeatureSchema `json:"features"`
}

type FeatureSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// EntitySchema represents the expected JSON structure from StageLevel2Entity
type EntityListSchema struct {
	Entities []EntitySchema `json:"entities"`
}

type EntitySchema struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // e.g., "struct", "func", "interface"
	Description string `json:"description"`
}

// InteractionSchema represents the expected JSON structure from StageLevel3Feature (Graph)
type InteractionListSchema struct {
	Interactions []InteractionSchema `json:"interactions"`
}

type InteractionSchema struct {
	ToFeatureID int64  `json:"to_feature_id"`
	Description string `json:"description"`
}

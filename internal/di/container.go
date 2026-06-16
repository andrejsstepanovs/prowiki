package di

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/andrejsstepanovs/prowiki/internal/api"
	"github.com/andrejsstepanovs/prowiki/internal/ast"
	"github.com/andrejsstepanovs/prowiki/internal/db"
	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"github.com/andrejsstepanovs/prowiki/internal/handlers"
	"github.com/andrejsstepanovs/prowiki/internal/llm"
	"github.com/andrejsstepanovs/prowiki/internal/prompt"
	"github.com/andrejsstepanovs/prowiki/internal/queue"
	"github.com/andrejsstepanovs/prowiki/internal/scanner"
	"github.com/andrejsstepanovs/prowiki/internal/app/ingest"
	"github.com/andrejsstepanovs/prowiki/internal/store"
	"github.com/andrejsstepanovs/prowiki/internal/worker"

	go_litellm "github.com/andrejsstepanovs/go-litellm/client"
	"github.com/andrejsstepanovs/go-litellm/conf/connections/litellm"
)

type Container struct {
	DB               *sql.DB
	Completer        domain.Completer
	IngestionService *ingest.Service
	Daemon           *worker.Daemon
	Server           *api.Server
	Project          *domain.Project
}

func NewContainer(ctx context.Context, projectRoot string) (*Container, error) {
	dbPath := filepath.Join(projectRoot, ".prowiki.db")
	database, err := db.Open(db.Config{Path: dbPath})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	pStore := store.NewProjectStore(database)
	fStore := store.NewFileStore(database)
	vStore := store.NewFileVersionStore(database)
	jStore := store.NewJobStore(database)
	featStore := store.NewFeatureStore(database)
	dlqStore := store.NewDLQStore(database)

	projectName := filepath.Base(projectRoot)
	var project domain.Project
	err = database.QueryRowContext(ctx, "SELECT id, name FROM projects WHERE name = ?", projectName).Scan(&project.ID, &project.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			project = domain.Project{Name: projectName}
			if err := pStore.Create(ctx, &project); err != nil {
				return nil, fmt.Errorf("failed to create project: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load project: %w", err)
		}
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("WARN: OPENAI_API_KEY is not set. LLM features will fail.")
	}

	baseURL, _ := url.Parse("https://api.openai.com/v1")
	conn := litellm.Connection{
		URL: *baseURL,
	}
	cfg := go_litellm.Config{
		APIKey:      apiKey,
		Temperature: 0.1,
	}
	llmClient, err := go_litellm.New(cfg, conn)
	if err != nil {
		fmt.Printf("failed to create litellm client: %v\n", err)
	}
	completer := llm.NewClient(llmClient)

	walker := scanner.NewDefaultWalker()
	parser := ast.NewHeuristicParser()

	ingestionService := ingest.NewService(database, walker, parser, fStore, vStore, jStore, project.ID, projectRoot)

	registry := prompt.NewHardcodedRegistry()
	parseHandler := handlers.NewParseHandler(completer, registry, vStore, featStore, jStore)
	dispatcher := handlers.NewDispatcher(parseHandler.Handle)

	sqliteQueue := queue.NewSQLiteQueue(database, jStore, dlqStore)
	daemon := worker.NewDaemon(sqliteQueue, dispatcher, 2*time.Second)

	apiServer := api.NewServer(8080, &project, pStore, fStore, vStore, featStore, jStore)

	return &Container{
		DB:               database,
		Completer:        completer,
		IngestionService: ingestionService,
		Daemon:           daemon,
		Server:           apiServer,
		Project:          &project,
	}, nil
}

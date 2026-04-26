// Command sdp-bd-suggest classifies a beads issue title+description
// using the MicroFirst kNN tier (bdseverity + bdtype).
//
// Usage:
//
//	sdp-bd-suggest --title="fix nil pointer" [--description="..."] \
//	               [--format=json|human] [--ollama-url=http://localhost:11434] \
//	               [--corpus-path=.beads/issues.jsonl]
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"sdp_dev/internal/inference/microfirst/bdseverity"
	"sdp_dev/internal/inference/microfirst/bdtype"
	"sdp_dev/internal/inference/microfirst/embed"
)

// config holds all CLI parameters.
type config struct {
	title       string
	description string
	format      string
	ollamaURL   string
	corpusPath  string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run parses args, validates, and executes. Returns exit code.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sdp-bd-suggest", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var cfg config
	fs.StringVar(&cfg.title, "title", "", "Issue title (required)")
	fs.StringVar(&cfg.description, "description", "", "Issue description (optional)")
	fs.StringVar(&cfg.format, "format", "json", "Output format: json or human")
	fs.StringVar(&cfg.ollamaURL, "ollama-url", "http://localhost:11434", "Ollama base URL")
	fs.StringVar(&cfg.corpusPath, "corpus-path", ".beads/issues.jsonl", "Path to corpus JSONL file")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if cfg.title == "" {
		fmt.Fprintln(stderr, "error: --title is required")
		fs.Usage()
		return 1
	}

	if err := runClassify(context.Background(), cfg, stdout); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

// runClassify orchestrates loading corpus, running classifiers, and rendering output.
func runClassify(ctx context.Context, cfg config, w io.Writer) error {
	// Load severity corpus. When corpus is small (≤30 items), LoadCorpus puts
	// everything in eval and returns an empty train slice. In that case we use
	// the eval slice as training data so the classifier still works.
	sevTrain, sevEval, err := bdseverity.LoadCorpus(cfg.corpusPath)
	if err != nil {
		return fmt.Errorf("load corpus: %w", err)
	}
	if len(sevTrain) == 0 {
		sevTrain = sevEval
	}

	// Load type corpus (separate call — different type).
	typeCorpus, err := bdtype.LoadCorpus(cfg.corpusPath)
	if err != nil {
		return fmt.Errorf("load type corpus: %w", err)
	}
	// Same fallback for small corpora.
	if len(typeCorpus.Train) == 0 {
		typeCorpus.Train = typeCorpus.Eval
	}

	// Build embedder.
	ollamaEmbedder := embed.NewOllamaEmbedder(cfg.ollamaURL)
	cachedEmbedder := embed.NewCachedEmbedder(ollamaEmbedder, 512)

	// Build severity classifier.
	sevClassifier, err := bdseverity.New(ctx, cachedEmbedder, sevTrain, 0)
	if err != nil {
		return fmt.Errorf("build severity classifier: %w", err)
	}

	// Build type classifier.
	typeClassifier, err := bdtype.NewBdTypeMicro(ctx, cachedEmbedder, typeCorpus.Train)
	if err != nil {
		return fmt.Errorf("build type classifier: %w", err)
	}

	// Run classifiers.
	sevInput := bdseverity.BdInput{Title: cfg.title, Description: cfg.description}
	sevResult, _, err := sevClassifier.Run(ctx, sevInput)
	if err != nil {
		return fmt.Errorf("severity classify: %w", err)
	}

	typeInput := bdtype.BdInput{Title: cfg.title, Description: cfg.description}
	typeResult, _, err := typeClassifier.Run(ctx, typeInput)
	if err != nil {
		return fmt.Errorf("type classify: %w", err)
	}

	// Render output.
	switch cfg.format {
	case "human":
		return renderHuman(w, cfg.title, sevResult, typeResult)
	default:
		return renderJSON(w, cfg.title, sevResult, typeResult)
	}
}

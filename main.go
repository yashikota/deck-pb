package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

var (
	configFile string
	profile    string
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "deck-pb",
	Short: "Add progress bars to Google Slides presentations",
	Long:  "deck-pb adds visual progress bar indicators to Google Slides presentations.",
}

var applyCmd = &cobra.Command{
	Use:   "apply DECK_FILE",
	Short: "Add progress bars to the presentation",
	Long:  "Delete existing progress bars and create new ones based on configuration.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		pid, err := resolvePresentationID(args[0])
		if err != nil {
			return err
		}

		cfg, err := LoadConfig(configFile)
		if err != nil {
			return err
		}
		if err := cfg.Progress.validate(); err != nil {
			return fmt.Errorf("invalid progress config: %w", err)
		}

		srv, err := NewSlidesService(ctx, profile)
		if err != nil {
			return err
		}

		if err := ApplyProgressBars(ctx, srv, pid, cfg.Progress); err != nil {
			return err
		}

		fmt.Println("Progress bars applied successfully.")
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete DECK_FILE",
	Short: "Remove progress bars from the presentation",
	Long:  "Remove all progress bar shapes from the presentation.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		pid, err := resolvePresentationID(args[0])
		if err != nil {
			return err
		}

		srv, err := NewSlidesService(ctx, profile)
		if err != nil {
			return err
		}

		return DeleteProgressBars(ctx, srv, pid)
	},
}

// resolvePresentationID reads the presentation ID from the markdown file's YAML frontmatter.
func resolvePresentationID(mdFile string) (string, error) {
	pid, err := readPresentationIDFromFrontmatter(mdFile)
	if err != nil {
		return "", fmt.Errorf("failed to read presentation ID from %s: %w", mdFile, err)
	}
	if pid == "" {
		return "", fmt.Errorf("presentation ID not found in frontmatter of %s", mdFile)
	}
	return pid, nil
}

// readPresentationIDFromFrontmatter extracts presentationID from a markdown file's YAML frontmatter.
func readPresentationIDFromFrontmatter(mdFile string) (string, error) {
	b, err := os.ReadFile(mdFile)
	if err != nil {
		return "", err
	}

	sep := []byte("---\n")
	if !bytes.HasPrefix(b, sep) {
		return "", nil
	}
	rest := bytes.TrimPrefix(b, sep)
	parts := bytes.SplitN(rest, sep, 2)
	if len(parts) < 2 {
		return "", nil
	}

	var fm struct {
		PresentationID string `yaml:"presentationID"`
	}
	if err := yaml.Unmarshal(parts[0], &fm); err != nil {
		return "", nil // frontmatter parse failure is not fatal
	}
	return fm.PresentationID, nil
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "deck-pb.yml", "config file path")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "deck authentication profile")

	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(deleteCmd)
}

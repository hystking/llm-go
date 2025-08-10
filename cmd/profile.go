package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"llmx/pkg/config"

	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage llmx profiles",
}

var profileEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the profiles config in your $EDITOR",
	Run: func(cmd *cobra.Command, args []string) {
		// Resolve config path (respect --config, else default)
		cfgPath := configPath
		if cfgPath == "" {
			p, err := config.DefaultPath()
			if err != nil {
				fmt.Println("failed to resolve user config dir:", err)
				os.Exit(1)
			}
			cfgPath = p
		}

		// Ensure directory
		if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
			fmt.Println("failed to create config directory:", err)
			os.Exit(1)
		}

		// Create file with minimal template if not exists
		if _, err := os.Stat(cfgPath); err != nil {
			if os.IsNotExist(err) {
				tmpl := []byte("{\n  \"default_profile\": \"\",\n  \"profiles\": {}\n}\n")
				if err := os.WriteFile(cfgPath, tmpl, 0o644); err != nil {
					fmt.Println("failed to create config file:", err)
					os.Exit(1)
				}
			} else {
				fmt.Println("failed to stat config file:", err)
				os.Exit(1)
			}
		}

		// Open editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		c := exec.Command(editor, cfgPath)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			fmt.Printf("failed to open editor %q: %v\n", editor, err)
			os.Exit(1)
		}
	},
}

func init() {
	// Reuse global --config flag binding
	profileEditCmd.Flags().StringVar(&configPath, "config", "", "path to config file (defaults to ~/.config/llmx/config.json)")
	profileCmd.AddCommand(profileEditCmd)
	rootCmd.AddCommand(profileCmd)
}

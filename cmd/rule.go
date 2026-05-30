package cmd

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/jlavera/mf-cli/internal/rule"
	"github.com/spf13/cobra"
)

var ruleCopy bool

var ruleCmd = &cobra.Command{
	Use:   "rule",
	Short: "Print the mf usage guide for AI agents (use --copy for clipboard)",
	Long: `Print or copy a ready-to-paste rule that teaches an AI agent how to use mf
in this project. Paste it into your assistant's setup — a Cursor rule
(.cursor/rules/), AGENTS.md, CLAUDE.md, or any system/project prompt.

By default the rule is printed to stdout (handy for piping into a file). Use
--copy to send it to your clipboard instead.`,
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		content := rule.Content

		if !ruleCopy {
			fmt.Print(content)
			return nil
		}

		if err := clipboard.WriteAll(content); err != nil {
			// Clipboard unavailable (e.g. headless or no pbcopy/xclip) —
			// fall back to stdout so the user still gets the content.
			fmt.Fprintln(os.Stderr, "⚠️  Could not access the clipboard, printing instead:")
			fmt.Print(content)
			return nil
		}

		fmt.Println("✅ Copied to clipboard — paste it into your AI assistant's rules/setup")
		return nil
	},
}

func init() {
	ruleCmd.Flags().BoolVarP(&ruleCopy, "copy", "c", false, "copy the rule to the clipboard instead of printing")
	ruleCmd.GroupID = "general"
	rootCmd.AddCommand(ruleCmd)
}

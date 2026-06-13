package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) subjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subjects",
		Short: "List available eLife subject areas",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a.progressf("fetching subjects...")
			subjects, err := a.client.Subjects(cmd.Context())
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(subjects, len(subjects))
		},
	}
	return cmd
}

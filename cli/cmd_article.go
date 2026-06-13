package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) articleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "article <id>",
		Short: "Get a single eLife article by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			a.progressf("fetching article %s...", id)
			art, err := a.client.Article(cmd.Context(), id)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render(art)
		},
	}
	return cmd
}

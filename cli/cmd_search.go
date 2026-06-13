package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/elife-cli/elife"
)

func (a *App) searchCmd() *cobra.Command {
	var (
		page  int
		order string
	)
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search eLife articles by keyword",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(20)
			opts := elife.SearchOptions{
				Query:   args[0],
				PerPage: n,
				Page:    page,
				Order:   order,
			}
			a.progressf("searching for %q...", args[0])
			items, _, err := a.client.Search(cmd.Context(), opts)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(items, len(items))
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().StringVar(&order, "order", "desc", "result order: asc or desc")
	return cmd
}

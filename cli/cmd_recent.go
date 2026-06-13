package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/elife-cli/elife"
)

func (a *App) recentCmd() *cobra.Command {
	var (
		page    int
		subject string
		artType string
		order   string
	)
	cmd := &cobra.Command{
		Use:   "recent",
		Short: "List recent eLife articles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			opts := elife.ListOptions{
				PerPage: n,
				Page:    page,
				Order:   order,
				Subject: subject,
				Type:    artType,
			}
			a.progressf("fetching %d recent articles...", n)
			items, _, err := a.client.Recent(cmd.Context(), opts)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(items, len(items))
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().StringVar(&subject, "subject", "", "filter by subject id (e.g. cell-biology)")
	cmd.Flags().StringVar(&artType, "type", "", "filter by article type (e.g. research-article)")
	cmd.Flags().StringVar(&order, "order", "desc", "publication order: asc or desc")
	return cmd
}

package cli

import (
	"errors"

	"github.com/tamnd/elife-cli/elife"
)

func isNotFound(err error) bool {
	return errors.Is(err, elife.ErrNotFound)
}

package cli

import (
	"github.com/voidarchive/Gator/internal/config"
	"github.com/voidarchive/Gator/internal/database"
)

type State struct {
	Cfg *config.Config
	DB  *database.Queries
}

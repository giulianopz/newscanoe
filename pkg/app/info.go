package app

import (
	_ "embed"
)

const Name = "newscanoe"

//go:generate bash version.sh
//go:embed version.txt
var Version string

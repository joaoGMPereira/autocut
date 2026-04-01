package seed

import "embed"

//go:embed assets/seed_config.json
//go:embed assets/client_secret_inerd.json
//go:embed assets/client_secret_maromba.json
//go:embed assets/client_secret_react.json
//go:embed assets/music/*.mp3
var assets embed.FS

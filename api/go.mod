module github.com/finkf/pcwgo/api

require (
	github.com/finkf/pcwgo/database v0.0.1
	github.com/finkf/pcwgo/database/sqlite v0.0.1
	github.com/finkf/pcwgo/database/user v0.0.1
)

replace (
	github.com/finkf/pcwgo/database => ../database
	github.com/finkf/pcwgo/database/sqlite => ../database/sqlite
	github.com/finkf/pcwgo/database/user => ../database/user
)

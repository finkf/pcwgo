module github.com/finkf/pcwgo/database/session

require (
	github.com/finkf/pcwgo/database v0.0.1
	github.com/finkf/pcwgo/database/sqlite v0.0.1
	github.com/finkf/pcwgo/database/user v0.0.1
)

replace github.com/finkf/pcwgo/database => ../

replace github.com/finkf/pcwgo/database/user => ../user

replace github.com/finkf/pcwgo/database/sqlite => ../sqlite

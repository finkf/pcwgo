module github.com/finkf/pcwgo/database/project

require github.com/finkf/pcwgo/database/user v0.0.1

require github.com/finkf/pcwgo/database v0.0.1

require (
	github.com/finkf/logger v0.3.2
	github.com/finkf/pcwgo/database/sqlite v0.0.1
)

replace github.com/finkf/pcwgo/database/user => ../user

replace github.com/finkf/pcwgo/database => ../

replace github.com/finkf/pcwgo/database/sqlite => ../sqlite

module github.com/finkf/pcwgo/database/user

require (
	github.com/finkf/logger v0.3.2
	github.com/finkf/pcwgo/database v0.0.1
	github.com/finkf/pcwgo/database/sqlite v0.0.1
	golang.org/x/crypto v0.0.0-20181012144002-a92615f3c490
	rsc.io/sqlite v0.0.0-20151027002647-c7a7bd4dbacb
)

replace github.com/finkf/pcwgo/database => ../

replace github.com/finkf/pcwgo/database/sqlite => ../sqlite

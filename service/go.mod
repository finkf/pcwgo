module github.com/finkf/pcwgo/service

go 1.12

require (
	github.com/apex/log v1.1.0
	github.com/bluele/gcache v0.0.0-20190301044115-79ae3b2d8680
	github.com/finkf/pcwgo/api v0.6.0
	github.com/finkf/pcwgo/db v0.9.0
	github.com/pkg/errors v0.8.1 // indirect
	github.com/sirupsen/logrus v1.4.1
)

replace github.com/finkf/pcwgo/db => ../db

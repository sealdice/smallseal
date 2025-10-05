package types

// 可能这个包叫做 core 更合适？

import "github.com/Masterminds/semver/v3"

var (
	APPNAME = "SealDice"
	VERSION = semver.MustParse(VERSION_MAIN + VERSION_PRERELEASE + VERSION_BUILD_METADATA)

	// VERSION_MAIN 主版本号
	VERSION_MAIN = "2.0.0"
	// VERSION_PRERELEASE 先行版本号
	VERSION_PRERELEASE = "-alpha"
	// VERSION_BUILD_METADATA 版本编译信息
	VERSION_BUILD_METADATA = ""

	// APP_CHANNEL 更新频道，stable/dev，在 action 构建时自动注入
	APP_CHANNEL = "dev" //nolint:revive
)

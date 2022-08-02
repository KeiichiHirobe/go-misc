//go:generate mockgen -source=$GOFILE -destination=./mock_$GOFILE -package=$GOPACKAGE

package main

type FeatureConfig interface {
	// プロジェクト名 + ONでフラグ名とする
	ProjectAOn() bool
	ProjectBOn(userID uint64) bool
}

// 設定ファイルを元に環境変数が設定されているので、環境変数からフラグ設定を読み取る
type envConfig struct {
	ProjectAOn bool `required:"true" envconfig:"ProjectAOn" default:"false"`
	ProjectBOn bool `required:"true" envconfig:"ProjectBOn" default:"false"`
}

type featureConfig struct {
	envConfig envConfig
}

func (f featureConfig) ProjectAOn() bool {
	return f.envConfig.ProjectAOn
}

func (f featureConfig) ProjectBOn(userID uint64) bool {
	var whiteListUserIDs = []uint64{100, 111}
	for _, wuid := range whiteListUserIDs {
		if wuid == userID {
			return true
		}
	}
	return f.envConfig.ProjectBOn
}

func NewFeatureConfig(envConfig envConfig) FeatureConfig {
	return &featureConfig{envConfig: envConfig}
}

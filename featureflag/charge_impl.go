package main

// 本当はinterfaceは別ファイルに定義されている想定だが、便宜上同じファイルに定義
type Charge interface {
	Charge(userID uint64, amount uint64)
}

type chargeService struct {
	featureConfig FeatureConfig
}

func (s *chargeService) Charge(userID uint64, amount uint64) {
	if !s.featureConfig.ProjectBOn(userID) {
		// 既存処理はここ
	} else {
		//
	}

	// charge実行
}

// DIツールで依存注入
// テストはfeatureConfigのmockを使えばいい
func NewChargeService(featureConfig FeatureConfig) Charge {
	return &chargeService{featureConfig: featureConfig}
}

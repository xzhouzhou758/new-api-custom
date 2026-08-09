package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// CheckinSetting 签到功能配置
type CheckinSetting struct {
	Enabled  bool `json:"enabled"`   // 是否启用签到功能
	MinQuota int  `json:"min_quota"` // 签到最小额度奖励
	MaxQuota int  `json:"max_quota"` // 签到最大额度奖励
	// 自定义规则：签到将额度补充到 TopUpTargetQuota；仅当额度低于 MinQuotaForCheckin 时才能签到
	TopUpTargetQuota   int `json:"top_up_target_quota"`   // 签到补充到的目标额度
	MinQuotaForCheckin int `json:"min_quota_for_checkin"` // 低于该额度才允许签到
}

// 默认配置
var checkinSetting = CheckinSetting{
	Enabled:            false, // 默认关闭
	MinQuota:           1000,  // 默认最小额度 1000 (约 0.002 USD)
	MaxQuota:           10000, // 默认最大额度 10000 (约 0.02 USD)
	TopUpTargetQuota:   200,   // 签到补充到的目标额度
	MinQuotaForCheckin: 10,    // 低于该额度才能签到
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("checkin_setting", &checkinSetting)
}

// GetCheckinSetting 获取签到配置
func GetCheckinSetting() *CheckinSetting {
	return &checkinSetting
}

// IsCheckinEnabled 是否启用签到功能
func IsCheckinEnabled() bool {
	return checkinSetting.Enabled
}

// GetCheckinQuotaRange 获取签到额度范围
func GetCheckinQuotaRange() (min, max int) {
	return checkinSetting.MinQuota, checkinSetting.MaxQuota
}

package system_setting

import "github.com/QuantumNous/new-api/setting/config"

type DiscordSettings struct {
	Enabled      bool   `json:"enabled"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	// GuildVerifyEnabled 是否启用 Discord 服务器/身份组校验
	GuildVerifyEnabled bool `json:"guild_verify_enabled"`
	// RequiredGuildId 只允许该服务器的成员登录
	RequiredGuildId string `json:"required_guild_id"`
	// RequiredRoleIds 只允许持有这些身份组（ID 逗号分隔）的成员登录，留空则仅要求是服务器成员
	RequiredRoleIds string `json:"required_role_ids"`
}

// 默认配置
var defaultDiscordSettings = DiscordSettings{}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("discord", &defaultDiscordSettings)
}

func GetDiscordSettings() *DiscordSettings {
	return &defaultDiscordSettings
}

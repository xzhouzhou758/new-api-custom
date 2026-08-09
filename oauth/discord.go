package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func init() {
	Register("discord", &DiscordProvider{})
}

// DiscordProvider implements OAuth for Discord
type DiscordProvider struct{}

type discordOAuthResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type discordUser struct {
	UID    string `json:"id"`
	ID     string `json:"username"`
	Name   string `json:"global_name"`
	Avatar string `json:"avatar"`
}

func (p *DiscordProvider) GetName() string {
	return "Discord"
}

func (p *DiscordProvider) IsEnabled() bool {
	return system_setting.GetDiscordSettings().Enabled
}

func (p *DiscordProvider) ExchangeToken(ctx context.Context, code string, c *gin.Context) (*OAuthToken, error) {
	if code == "" {
		return nil, NewOAuthError(i18n.MsgOAuthInvalidCode, nil)
	}

	logger.LogDebug(ctx, "[OAuth-Discord] ExchangeToken: code=%s...", code[:min(len(code), 10)])

	settings := system_setting.GetDiscordSettings()
	redirectUri := fmt.Sprintf("%s/oauth/discord", system_setting.ServerAddress)
	values := url.Values{}
	values.Set("client_id", settings.ClientId)
	values.Set("client_secret", settings.ClientSecret)
	values.Set("code", code)
	values.Set("grant_type", "authorization_code")
	values.Set("redirect_uri", redirectUri)

	logger.LogDebug(ctx, "[OAuth-Discord] ExchangeToken: redirect_uri=%s", redirectUri)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://discord.com/api/v10/oauth2/token", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Discord] ExchangeToken error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "Discord"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-Discord] ExchangeToken response status: %d", res.StatusCode)

	var discordResponse discordOAuthResponse
	err = json.NewDecoder(res.Body).Decode(&discordResponse)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Discord] ExchangeToken decode error: %s", err.Error()))
		return nil, err
	}

	if discordResponse.AccessToken == "" {
		logger.LogError(ctx, "[OAuth-Discord] ExchangeToken failed: empty access token")
		return nil, NewOAuthError(i18n.MsgOAuthTokenFailed, map[string]any{"Provider": "Discord"})
	}

	logger.LogDebug(ctx, "[OAuth-Discord] ExchangeToken success: scope=%s", discordResponse.Scope)

	return &OAuthToken{
		AccessToken:  discordResponse.AccessToken,
		TokenType:    discordResponse.TokenType,
		RefreshToken: discordResponse.RefreshToken,
		ExpiresIn:    discordResponse.ExpiresIn,
		Scope:        discordResponse.Scope,
		IDToken:      discordResponse.IDToken,
	}, nil
}

func (p *DiscordProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*OAuthUser, error) {
	logger.LogDebug(ctx, "[OAuth-Discord] GetUserInfo: fetching user info")

	req, err := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Discord] GetUserInfo error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "Discord"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-Discord] GetUserInfo response status: %d", res.StatusCode)

	if res.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Discord] GetUserInfo failed: status=%d", res.StatusCode))
		return nil, NewOAuthError(i18n.MsgOAuthGetUserErr, nil)
	}

	var discordUser discordUser
	err = json.NewDecoder(res.Body).Decode(&discordUser)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Discord] GetUserInfo decode error: %s", err.Error()))
		return nil, err
	}

	if discordUser.UID == "" || discordUser.ID == "" {
		logger.LogError(ctx, "[OAuth-Discord] GetUserInfo failed: empty user fields")
		return nil, NewOAuthError(i18n.MsgOAuthUserInfoEmpty, map[string]any{"Provider": "Discord"})
	}

	// 若启用了服务器/身份组校验，则验证用户是否属于指定服务器并持有允许的身份组
	if system_setting.GetDiscordSettings().GuildVerifyEnabled {
		if err := verifyDiscordGuildMembership(ctx, token); err != nil {
			return nil, err
		}
	}

	logger.LogDebug(ctx, "[OAuth-Discord] GetUserInfo success: uid=%s, username=%s, name=%s", discordUser.UID, discordUser.ID, discordUser.Name)

	avatarURL := ""
	if discordUser.Avatar != "" {
		ext := "png"
		if strings.HasPrefix(discordUser.Avatar, "a_") {
			ext = "gif"
		}
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s?size=256", discordUser.UID, discordUser.Avatar, ext)
	}

	return &OAuthUser{
		ProviderUserID: discordUser.UID,
		// 使用 Discord 用户 ID（数字）作为站点 username，注册时不可修改且全局唯一
		Username:    discordUser.UID,
		DisplayName: discordUser.Name,
		Extra: map[string]any{
			"avatar": avatarURL,
		},
	}, nil
}

func (p *DiscordProvider) IsUserIDTaken(providerUserID string) bool {
	return model.IsDiscordIdAlreadyTaken(providerUserID)
}

func (p *DiscordProvider) FillUserByProviderID(user *model.User, providerUserID string) error {
	user.DiscordId = providerUserID
	return user.FillUserByDiscordId()
}

func (p *DiscordProvider) SetProviderUserID(user *model.User, providerUserID string) {
	user.DiscordId = providerUserID
}

func (p *DiscordProvider) GetProviderPrefix() string {
	return "discord_"
}

// discordGuild is a minimal representation of a guild returned by
// GET /users/@me/guilds.
type discordGuild struct {
	ID string `json:"id"`
}

// discordGuildMember is a minimal representation of the member object returned
// by GET /users/@me/guilds/{guild.id}/member.
type discordGuildMember struct {
	Roles []string `json:"roles"`
}

// verifyDiscordGuildMembership verifies, using the user's OAuth access token,
// that the user is a member of the required guild and (optionally) holds at
// least one of the required role IDs.
// This requires the `guilds` and `guilds.members.read` OAuth scopes.
func verifyDiscordGuildMembership(ctx context.Context, token *OAuthToken) error {
	settings := system_setting.GetDiscordSettings()
	guildID := strings.TrimSpace(settings.RequiredGuildId)
	if guildID == "" {
		return &AccessDeniedError{Message: "guild ID is required for role verification"}
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	// 1. Confirm the user is a member of the required guild
	req, err := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/v10/users/@me/guilds", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Discord] GetUserGuilds error: %s", err.Error()))
		return NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "Discord"}, err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Discord] GetUserGuilds returned status %d", res.StatusCode))
		return NewOAuthErrorWithRaw(i18n.MsgOAuthGetUserErr, nil, fmt.Sprintf("Discord API returned status %d for GetUserGuilds", res.StatusCode))
	}

	var guilds []discordGuild
	if err := json.NewDecoder(res.Body).Decode(&guilds); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Discord] GetUserGuilds parse error: %s", err.Error()))
		return err
	}

	inGuild := false
	for _, g := range guilds {
		if g.ID == guildID {
			inGuild = true
			break
		}
	}
	if !inGuild {
		logger.LogWarn(ctx, fmt.Sprintf("[OAuth-Discord] role verification failed: user is not a member of guild %s", guildID))
		return &AccessDeniedError{Message: "Discord role verification error: you are not a member of the required server. / 你没有访问权限：你不是指定 Discord 服务器的成员。"}
	}

	// 2. Fetch the user's member info in the guild to read their roles
	req2, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://discord.com/api/v10/users/@me/guilds/%s/member", guildID), nil)
	if err != nil {
		return err
	}
	req2.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req2.Header.Set("Accept", "application/json")

	res2, err := client.Do(req2)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Discord] GetUserGuildMember error: %s", err.Error()))
		return NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "Discord"}, err.Error())
	}
	defer res2.Body.Close()

	if res2.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Discord] GetUserGuildMember returned status %d", res2.StatusCode))
		return NewOAuthErrorWithRaw(i18n.MsgOAuthGetUserErr, nil, fmt.Sprintf("Discord API returned status %d for GetUserGuildMember", res2.StatusCode))
	}

	var member discordGuildMember
	if err := json.NewDecoder(res2.Body).Decode(&member); err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Discord] GetUserGuildMember parse error: %s", err.Error()))
		return err
	}

	// 3. Check the required roles if any
	requiredRoles := strings.TrimSpace(settings.RequiredRoleIds)
	if requiredRoles == "" {
		// 仅要求是服务器成员
		return nil
	}

	roleSet := make(map[string]bool, len(member.Roles))
	for _, r := range member.Roles {
		roleSet[r] = true
	}
	for _, r := range strings.Split(requiredRoles, ",") {
		if r = strings.TrimSpace(r); r != "" && roleSet[r] {
			return nil
		}
	}

	logger.LogWarn(ctx, fmt.Sprintf("[OAuth-Discord] role verification failed: user holds none of the required roles in guild %s", guildID))
	return &AccessDeniedError{Message: "Discord role verification error: you do not hold any of the required roles. / 你没有访问权限：你需要拥有指定 Discord 服务器中的相应身份组。"}
}

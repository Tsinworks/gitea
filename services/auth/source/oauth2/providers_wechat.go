// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package oauth2

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"

	"code.gitea.io/gitea/modules/svg"

	"github.com/markbates/goth"
	"golang.org/x/oauth2"
)

// WeChatProvider is a GothProvider for WeChat
type WeChatProvider struct {
	BaseProvider
	scopes []string
}

// WeChatUser represents WeChat user information
type WeChatUser struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	Nickname   string `json:"nickname"`
	Sex        int    `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	HeadImgURL string `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
}

// WeChatTokenResponse represents WeChat access token response
type WeChatTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

// NewWeChatProvider creates a new WeChat provider
func NewWeChatProvider(name, displayName string, scopes []string) *WeChatProvider {
	return &WeChatProvider{
		BaseProvider: BaseProvider{
			name:        name,
			displayName: displayName,
		},
		scopes: scopes,
	}
}

// Name returns the provider name
func (p *WeChatProvider) Name() string {
	return p.name
}

// DisplayName returns the display name
func (p *WeChatProvider) DisplayName() string {
	return p.displayName
}

// IconHTML returns the icon HTML for WeChat
func (p *WeChatProvider) IconHTML(size int) template.HTML {
	svgHTML := svg.RenderHTML("gitea-wechat", size, "tw-mr-2")
	if svgHTML == "" {
		// Fallback to OpenID icon if WeChat icon is not available
		svgHTML = svg.RenderHTML("gitea-openid", size, "tw-mr-2")
	}
	return svgHTML
}

// CustomURLSettings returns nil as WeChat doesn't use custom URLs
func (p *WeChatProvider) CustomURLSettings() *CustomURLSettings {
	return nil
}

// CreateGothProvider creates a GothProvider from this Provider
func (p *WeChatProvider) CreateGothProvider(providerName, callbackURL string, source *Source) (goth.Provider, error) {
	scopes := make([]string, len(p.scopes)+len(source.Scopes))
	copy(scopes, p.scopes)
	copy(scopes[len(p.scopes):], source.Scopes)
	
	return &weChatGothProvider{
		ClientKey:   source.ClientID,
		Secret:      source.ClientSecret,
		CallbackURL: callbackURL,
		HTTPClient:  &http.Client{},
		config: &oauth2.Config{
			ClientID:     source.ClientID,
			ClientSecret: source.ClientSecret,
			RedirectURL:  callbackURL,
			Scopes:       scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://open.weixin.qq.com/connect/oauth2/authorize",
				TokenURL: "https://api.weixin.qq.com/sns/oauth2/access_token",
			},
		},
	}, nil
}

// weChatGothProvider implements goth.Provider for WeChat
type weChatGothProvider struct {
	ClientKey   string
	Secret      string
	CallbackURL string
	HTTPClient  *http.Client
	config      *oauth2.Config
	providerName string
}

// Name returns the provider name
func (p *weChatGothProvider) Name() string {
	return p.providerName
}

// SetName sets the provider name
func (p *weChatGothProvider) SetName(name string) {
	p.providerName = name
}

// Debug sets debug mode (not implemented for WeChat)
func (p *weChatGothProvider) Debug(debug bool) {
	// WeChat provider doesn't support debug mode
}

// BeginAuth starts the WeChat OAuth flow
func (p *weChatGothProvider) BeginAuth(state string) (goth.Session, error) {
	return &weChatSession{
		AuthURL: p.config.AuthCodeURL(state),
		State:   state,
	}, nil
}

// UnmarshalSession unmarshals a WeChat session
func (p *weChatGothProvider) UnmarshalSession(data string) (goth.Session, error) {
	sess := &weChatSession{}
	err := json.Unmarshal([]byte(data), sess)
	return sess, err
}

// FetchUser fetches user information from WeChat
func (p *weChatGothProvider) FetchUser(session goth.Session) (goth.User, error) {
	sess := session.(*weChatSession)
	
	// Get user info from WeChat API
	userURL := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN", 
		sess.AccessToken, sess.OpenID)
	
	resp, err := p.HTTPClient.Get(userURL)
	if err != nil {
		return goth.User{}, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return goth.User{}, err
	}
	
	var weChatUser WeChatUser
	if err := json.Unmarshal(body, &weChatUser); err != nil {
		return goth.User{}, err
	}
	
	user := goth.User{
		UserID:      weChatUser.OpenID,
		Name:        weChatUser.Nickname,
		NickName:    weChatUser.Nickname,
		AvatarURL:   weChatUser.HeadImgURL,
		Location:    fmt.Sprintf("%s, %s, %s", weChatUser.Country, weChatUser.Province, weChatUser.City),
		AccessToken: sess.AccessToken,
		Provider:    p.Name(),
		RawData:     map[string]interface{}{
			"openid":     weChatUser.OpenID,
			"unionid":    weChatUser.UnionID,
			"nickname":   weChatUser.Nickname,
			"sex":        weChatUser.Sex,
			"province":   weChatUser.Province,
			"city":       weChatUser.City,
			"country":    weChatUser.Country,
			"headimgurl": weChatUser.HeadImgURL,
			"privilege":  weChatUser.Privilege,
		},
	}
	
	return user, nil
}

// RefreshTokenAvailable returns false as WeChat doesn't support refresh tokens in this implementation
func (p *weChatGothProvider) RefreshTokenAvailable() bool {
	return false
}

// RefreshToken refreshes the access token (not implemented for WeChat)
func (p *weChatGothProvider) RefreshToken(refreshToken string) (*oauth2.Token, error) {
	return nil, fmt.Errorf("refresh token not supported for WeChat provider")
}

// weChatSession represents a WeChat OAuth session
type weChatSession struct {
	AuthURL      string
	AccessToken  string
	RefreshToken string
	OpenID       string
	State        string
}

// GetAuthURL returns the auth URL for WeChat
func (s *weChatSession) GetAuthURL() (string, error) {
	if s.AuthURL == "" {
		return "", fmt.Errorf("no auth URL available")
	}
	return s.AuthURL, nil
}

// Authorize completes the WeChat OAuth flow
func (s *weChatSession) Authorize(provider goth.Provider, params goth.Params) (string, error) {
	p := provider.(*weChatGothProvider)
	
	code := params.Get("code")
	if code == "" {
		return "", fmt.Errorf("missing code parameter")
	}
	
	// Exchange code for access token
	tokenURL := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		p.ClientKey, p.Secret, code)
	
	resp, err := p.HTTPClient.Get(tokenURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	var tokenResp WeChatTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	
	if tokenResp.ErrCode != 0 {
		return "", fmt.Errorf("WeChat API error: %s", tokenResp.ErrMsg)
	}
	
	s.AccessToken = tokenResp.AccessToken
	s.RefreshToken = tokenResp.RefreshToken
	s.OpenID = tokenResp.OpenID
	
	return tokenResp.AccessToken, nil
}

// Marshal marshals the WeChat session
func (s *weChatSession) Marshal() string {
	j, _ := json.Marshal(s)
	return string(j)
}

var _ GothProvider = &WeChatProvider{} 
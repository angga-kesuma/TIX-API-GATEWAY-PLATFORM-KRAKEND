package outbound

import (
	"context"
	"net/http"
	"sync"

	"github.com/tiket/TIX-API-GATEWAY-PLATFORM-KRAKEND/pkg/helper/httpclient"
	"github.com/tiket/TIX-API-GATEWAY-PLATFORM-KRAKEND/pkg/plugin/config"
)

const (
	SessionValidatePath = "/v1/session/validate"
)

type (
	OutboundMemberSession struct {
		httpClient *http.Client
		cfg        *config.AppConfig
	}
)

var (
	instance *OutboundMemberSession
	once     sync.Once
)

// GetOutboundMemberSession returns a singleton instance of OutboundMemberSession
// initialized lazily using sync.Once. The entire struct is created only once.
func GetOutboundMemberSession(cfg *config.AppConfig) *OutboundMemberSession {
	once.Do(func() {
		instance = &OutboundMemberSession{
			httpClient: httpclient.NewHTTPClient(cfg.MemberSession.HTTPConfig),
			cfg:        cfg,
		}
	})
	return instance
}

func (o *OutboundMemberSession) ValidateSession(ctx context.Context, accountIds []int64) (MemberAuthSessionData, error) {

	// Build request to member session validation endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.MemberSession.BaseUrl+SessionValidatePath, nil)
	if err != nil {
		return MemberAuthSessionData{}, err
	}

	// Execute request using the lazy-initialized HTTP client
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return MemberAuthSessionData{}, err
	}
	defer resp.Body.Close()

	res := MemberAuthSessionResponse{}
	return res.Data, nil
}

type MemberAuthSessionResponse struct {
	Code       string                `json:"code"`
	Message    string                `json:"message"`
	Errors     any                   `json:"errors"`
	Data       MemberAuthSessionData `json:"data"`
	ServerTime string                `json:"serverTime"`
}

type MemberAuthSessionData struct {
	SessionId                string           `json:"sessionId"`
	AccountId                int              `json:"accountId"`
	UserName                 string           `json:"username"`
	FirstName                string           `json:"firstname"`
	LastName                 string           `json:"lastname"`
	Audience                 string           `json:"audience"`
	IsRefreshToken           bool             `json:"isRefreshToken"`
	DeviceId                 string           `json:"deviceId"`
	Language                 string           `json:"language"`
	Currency                 string           `json:"currency"`
	OldSessionId             string           `json:"oldSessionId"`
	OldSessionType           string           `json:"oldSessionType"`
	TixPoint                 int              `json:"tixPoint"`
	AccessToken              string           `json:"accessToken"`
	RefreshToken             string           `json:"refreshToken"`
	BusinessId               string           `json:"businessId"`
	B2BCorporateResponseData interface{}      `json:"b2bCorporateResponseData"`
	IsLogin                  bool             `json:"isLogin"`
	UnmUserId                int              `json:"unmUserId"`
	VerifiedEmail            bool             `json:"verifiedEmail"`
	MemberType               string           `json:"memberType,omitempty"`
	VerifiedPhoneNumber      bool             `json:"verifiedPhoneNumber"`
	LoginMedia               string           `json:"loginMedia"`
	LoyaltyLevel             loyaltyLevel     `json:"loyaltyLevel"`
	Priv                     string           `json:"priv"`
	Role                     string           `json:"role"`
	Group                    string           `json:"group"`
	Token                    string           `json:"token"`
	AdditionalData           []additionalData `json:"additionalData"`
}

type loyaltyLevel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type additionalData struct {
	Vertical    string                `json:"vertical"`
	Data        additionalDataDetails `json:"data"`
	SharedLevel string                `json:"sharedLevel"`
}

type additionalDataDetails struct {
	SelectedPartner selectedPartner `json:"selectedPartner"`
}

type selectedPartner struct {
	Priv       string `json:"priv"`
	Role       string `json:"role"`
	BusinessID string `json:"businessId"`
	Group      string `json:"group"`
}

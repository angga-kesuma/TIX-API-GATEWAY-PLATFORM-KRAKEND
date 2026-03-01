package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/samber/lo"
	"github.com/tiket/TIX-AFFILIATE-COMMON-GO/collection"
)

type (
	cookieType string
	cookieKey  string
	contextKey string
)

const (
	// notes:
	// stt stands for session token type, the goal is to differentiate session between account session (it could be b2c, b2b, and admin)
	// for further reading, please refer to this docs https://borobudur.atlassian.net/wiki/spaces/MEM/pages/3035004944/Separate+Session+Type+B2c+and+Admin

	b2cSTTParamValue                    = "0"
	adminSTTParamValue                  = "1"
	b2bSTTParamValue                    = "2"
	b2cCookieType            cookieType = "B2C"
	adminCookieType          cookieType = "ADMIN"
	b2bCookieType            cookieType = "B2B"
	sessionCookieKey         cookieKey  = "session_access_token"
	aatCookieKey             cookieKey  = "aat"
	memberAuthSessionDataKey contextKey = "member_auth_session_data"
	eventPartner                        = "event_partner"
	verticalTTD                         = "TTD"
	sharedLevelSession                  = "SESSION"
	sttKey                              = "stt"
	additionalDataKey                   = "additionalData"

	configKeyPrivilegeIds = "privilegeIds"
	configKeyAuthServer   = "authServer"
	configKeyTimeout      = "timeout"
	configKeyCookieType   = "cookieType"
)

var (
	memberAuthLogPrefix      = middlewareLogPrefix + "[member-auth]"
	errUnauthorized          = fmt.Errorf("unauthorized")
	cookieTypeToCookieKeyMap = map[cookieType]cookieKey{
		b2bCookieType:   aatCookieKey,
		b2cCookieType:   sessionCookieKey,
		adminCookieType: aatCookieKey,
	}
	cookieTypeToSTTParamValueMap = map[cookieType]string{
		b2cCookieType:   b2cSTTParamValue,
		b2bCookieType:   b2bSTTParamValue,
		adminCookieType: adminSTTParamValue,
	}
)

type memberAuthSessionResponse struct {
	Code       string                `json:"code"`
	Message    string                `json:"message"`
	Errors     any                   `json:"errors"`
	Data       memberAuthSessionData `json:"data"`
	ServerTime string                `json:"serverTime"`
}

type memberAuthSessionData struct {
	Username            string           `json:"username"`
	BusinessID          string           `json:"businessId"`
	AccountID           int64            `json:"accountId"`
	DeviceID            string           `json:"deviceId"`
	VerifiedPhoneNumber bool             `json:"verifiedPhoneNumber"`
	LoginMedia          string           `json:"loginMedia"`
	LoyaltyLevel        loyaltyLevel     `json:"loyaltyLevel"`
	Priv                string           `json:"priv"`
	Role                string           `json:"role"`
	Group               string           `json:"group"`
	Token               string           `json:"token"`
	AdditionalData      []additionalData `json:"additionalData"`
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

func (t cookieType) String() string {
	return string(t)
}

func (t cookieKey) String() string {
	return string(t)
}

func RegisterMemberAuth(next http.Handler, config map[string]any) (http.Handler, error) {
	isUsingPrivilege := false
	privilegeIds, ok := config[configKeyPrivilegeIds].([]any)
	if ok {
		isUsingPrivilege = true
	}

	privilegeIdSet := collection.NewSet[string]()
	if isUsingPrivilege {
		lo.ForEach(privilegeIds, func(i any, _ int) {
			privilegeIdSet.Add(i.(string))
		})
	}

	authServer, ok := config[configKeyAuthServer].(string)
	if !ok {
		panic("missing string config authServer")
	}

	timeoutStr, ok := config[configKeyTimeout].(string)
	if !ok {
		panic("missing string config timeout")
	}

	timeout, errParseDuration := time.ParseDuration(timeoutStr)
	if errParseDuration != nil {
		panic(fmt.Sprintf("failed to parse duration, error: %s", errParseDuration.Error()))
	}

	cookieTypeConfig, ok := config[configKeyCookieType].(string)
	if !ok || cookieTypeConfig == "" { // if empty, will assign with default value
		cookieTypeConfig = b2cCookieType.String()
	}

	cookieTypeFromConfig := cookieType(cookieTypeConfig)
	cookieKey, ok := cookieTypeToCookieKeyMap[cookieTypeFromConfig]
	if !ok {
		panic("invalid cookieType string")
	}

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var token string
		var authSource string
		reqPath := fmt.Sprintf("%s %s", request.Method, request.URL.Path)
		finalCookieType := cookieTypeFromConfig

		bearerToken, cType, err := getBearerToken(request.Header)
		if err == nil {
			token = bearerToken
			finalCookieType = cType
			authSource = "Bearer token"
		} else {
			tokenCookie, cookieErr := request.Cookie(cookieKey.String())
			if cookieErr != nil {
				logger.Error(memberAuthLogPrefix, fmt.Sprintf("failed to get token reqPath: %s, Bearer error: %s, Cookie error: %s",
					reqPath, err.Error(), cookieErr.Error()))
				writeJSONError(writer, http.StatusUnauthorized)

				return
			}

			token = fmt.Sprintf("Bearer %s", tokenCookie.Value)
			authSource = fmt.Sprintf("%s cookie", cookieKey.String())
		}

		// Validate the token
		memberAuthSessionResponse, err := getMemberAuthToken(request.Context(), timeout, authServer, token, reqPath, finalCookieType)
		if err != nil {
			if errors.Is(err, errUnauthorized) {
				writeJSONError(writer, http.StatusUnauthorized)

				return
			}

			logger.Error(memberAuthLogPrefix, fmt.Sprintf("failed to getMemberAuthToken from %s, error: %s", authSource, err.Error()))
			writeJSONError(writer, http.StatusInternalServerError)

			return
		}

		// TTD multiple partners feature use additional data
		additionalData := memberAuthSessionResponse.Data.AdditionalData
		if additionalData != nil && strings.EqualFold(request.Header.Get(headerKeyBusinessType), eventPartner) {
			for _, data := range additionalData {
				if data.Vertical == verticalTTD && data.SharedLevel == sharedLevelSession {
					selectedPartner := data.Data.SelectedPartner
					if selectedPartner.Priv != "" {
						memberAuthSessionResponse.Data.Priv = selectedPartner.Priv
					}
					if selectedPartner.Role != "" {
						memberAuthSessionResponse.Data.Role = selectedPartner.Role
					}
					if selectedPartner.BusinessID != "" {
						memberAuthSessionResponse.Data.BusinessID = selectedPartner.BusinessID
					}
					if selectedPartner.Group != "" {
						memberAuthSessionResponse.Data.Group = selectedPartner.Group
					}
				}
			}
		}

		if isUsingPrivilege {
			memberAuthPrivIds := strings.Split(memberAuthSessionResponse.Data.Priv, ",")
			_, found := lo.Find(memberAuthPrivIds, func(item string) bool {
				return privilegeIdSet.Contains(item)
			})

			if !found {
				logger.Error(memberAuthLogPrefix,
					fmt.Sprintf("the user has not required priv, memberAuthPrivIds: %v, privilegeIds: %v, reqPath: %s, cookieTypeFromConfig: %s",
						memberAuthPrivIds, privilegeIdSet.Values(), reqPath, cookieTypeFromConfig.String()))
				writeJSONError(writer, http.StatusUnauthorized)
				return
			}
		}

		memberAuthSessionResponse.Data.Token = token
		ctx := context.WithValue(request.Context(), memberAuthSessionDataKey, memberAuthSessionResponse.Data)
		request = request.WithContext(ctx)

		next.ServeHTTP(writer, request)
	}), nil
}

func getMemberAuthToken(ctx context.Context, timeout time.Duration, authServerURL, token, reqPath string,
	cookieTypeFromConfig cookieType) (*memberAuthSessionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authServerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to http.NewRequestWithContext, error: %s", err.Error())
	}

	uniqueID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to uuid.NewV4, error: %s", err.Error())
	}
	reqID := uniqueID.String()

	headers := http.Header{}
	headers.Add(headerKeyAuthorization, token)
	headers.Add(headerKeyStoreID, defaultStoreID)
	headers.Add(headerKeyRequestID, reqID)
	headers.Add(headerKeyChannelID, channelWebview)
	headers.Add(headerKeyUsername, defaultUsername)
	headers.Add(headerKeyServiceID, serviceID)
	headers.Add(headerKeyAudience, serviceAudience)

	req.Header = headers

	if sttValue, ok := cookieTypeToSTTParamValueMap[cookieTypeFromConfig]; ok {
		queryParams := url.Values{}
		queryParams.Add(sttKey, sttValue)
		queryParams.Add(additionalDataKey, verticalTTD)

		req.URL.RawQuery = queryParams.Encode()
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to http.DefaultClient.Do, reqID: %s, reqPath: %s, error: %s, headers: %v",
			reqID,
			reqPath,
			err.Error(),
			headers,
		)
	}

	defer func() {
		if bodyCloseError := res.Body.Close(); bodyCloseError != nil {
			logger.Error(memberAuthLogPrefix, fmt.Sprintf("failed to close body, reqID: %s, reqPath: %s, error: %s",
				reqID,
				reqPath,
				bodyCloseError.Error()),
			)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to io.ReadAll, reqID: %s, reqPath: %s, error: %s",
			reqID,
			reqPath,
			err.Error(),
		)
	}

	if res.StatusCode != http.StatusOK {
		logger.Error(memberAuthLogPrefix,
			fmt.Sprintf("failed to get 200 OK, reqID: %s, reqPath: %s, cookieType: %s, status: %d, headers: %v, body: %s",
				reqID,
				reqPath,
				cookieTypeFromConfig,
				res.StatusCode,
				headers,
				string(body),
			))

		return nil, errUnauthorized
	}

	if len(body) == 0 {
		logger.Error(memberAuthLogPrefix, fmt.Sprintf("empty body, reqID: %s, reqPath: %s, status: %d, headers: %v",
			reqID,
			reqPath,
			res.StatusCode,
			headers,
		))

		return nil, errUnauthorized
	}

	var memberAuthSessionResp memberAuthSessionResponse
	err = json.Unmarshal(body, &memberAuthSessionResp)
	if err != nil {
		return nil, fmt.Errorf("failed to json.Unmarshal, reqID: %s, reqPath: %s, error: %s, headers: %v, body: %s",
			reqID,
			reqPath,
			err.Error(),
			headers,
			string(body),
		)
	}

	if memberAuthSessionResp.Code != "SUCCESS" {
		logger.Error(memberAuthLogPrefix, fmt.Sprintf("failed to get SUCCESS, reqID: %s, reqPath: %s, code: %s, headers: %v, body: %s",
			reqID,
			reqPath,
			memberAuthSessionResp.Code,
			headers,
			string(body),
		))

		return nil, errUnauthorized
	}

	return &memberAuthSessionResp, nil
}

func getBearerToken(header http.Header) (string, cookieType, error) {
	bearerToken := header.Get(headerKeyAuthorization)
	if bearerToken == "" || !strings.HasPrefix(bearerToken, "Bearer ") {
		// there is a usecase that native apps use accessToken instead of bearer token, without Bearer prefix
		bearerToken = header.Get(headerKeyAccessToken)
		if bearerToken == "" {
			return "", "", fmt.Errorf("invalid bearer token")
		}
		bearerToken = fmt.Sprintf("Bearer %s", bearerToken)
	}

	jwt := strings.TrimSpace(strings.TrimPrefix(bearerToken, "Bearer "))
	split := strings.Split(jwt, ".")
	if len(split) != 3 {
		return "", "", fmt.Errorf("invalid JWT")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(split[1])
	if err != nil {
		return "", "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal JWT payload: %w", err)
	}

	cType := b2cCookieType
	stt := strconv.Itoa(payload.STT)
	switch stt {
	case adminSTTParamValue:
		cType = adminCookieType
	case b2bSTTParamValue:
		cType = b2bCookieType
	}

	return bearerToken, cType, nil
}

type jwtPayload struct {
	STT int `json:"stt"`
}

func writeJSONError(writer http.ResponseWriter, status int) {
	writer.Header().Set(headerKeyContentType, applicationJSONContentType)
	writer.WriteHeader(status)
	newJsonBodyWriter(status)(writer, nil, nil)
}

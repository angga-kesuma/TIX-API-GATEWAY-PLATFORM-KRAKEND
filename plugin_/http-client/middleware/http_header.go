package main

import (
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"

	gouseragent "github.com/medama-io/go-useragent"

	"github.com/gofrs/uuid"
)

const (
	serviceID       = "TTD_GATEWAY"
	serviceAudience = "tiket.com"

	headerKeyRequestID             = "X-Request-Id"
	headerKeyServiceID             = "X-Service-Id"
	headerKeyUsername              = "X-Username"
	headerKeyStoreID               = "X-Store-Id"
	headerKeyBusinessID            = "X-Business-Id"
	headerKeyAccountID             = "X-Account-Id"
	headerKeyLanguage              = "Accept-Language"
	headerKeyCurrency              = "X-Currency"
	headerKeyDeviceID              = "X-Device-Id"
	headerKeyIdentity              = "X-Identity"
	headerKeyResellerID            = "X-Reseller-Id"
	headerKeyResellerType          = "X-Reseller-Type"
	headerKeyLoyaltyLevel          = "X-Loyalty-Level"
	headerKeyChannelID             = "X-Channel-Id"
	headerKeyChannelIDV2           = "X-Channel-Id-V2"
	headerKeyTagAffId              = "Tag_Aff_Id"
	headerKeyForwardedFor          = "X-Forwarded-For"
	headerKeyTrueClientIP          = "True-Client-Ip"
	headerKeyProxyClientIP         = "Proxy-Client-IP"
	headerKeyRealIp                = "X-Real-Ip"
	headerKeyTwh                   = "twh"
	headerKeyPlatformV2            = "X-Platform-V2"
	headerKeyLoginMedia            = "X-Login-Media"
	headerKeyRoleID                = "X-Role-Id"
	headerKeyIsVerifiedPhoneNumber = "isVerifiedPhoneNumber"
	headerKeyUserAgent             = "User-Agent"
	headerKeyCookie                = "Cookie"
	headerKeySessionID             = "PHPSESSID"
	headerKeyBusinessType          = "X-Business-Type"
	headerKeyContentType           = "Content-Type"
	headerKeyAudience              = "X-Audience"

	// Old/Legacy Header Format
	oldHeaderKeyUsername   = "username"
	oldHeaderKeyRequestId  = "requestId"
	oldHeaderKeyStoreId    = "storeId"
	oldHeaderKeyCurrency   = "currency"
	oldHeaderKeyResellerId = "resellerId"
	oldHeaderKeyLang       = "lang"
	oldHeaderKeyLanguage   = "language"
	oldHeaderKeyDeviceID   = "deviceId"
	oldHeaderKeyUserAgent  = "userAgent"

	defaultUsername              = "guest"
	defaultStoreID               = "TIKETCOM"
	defaultUnknown               = "UNKNOWN"
	defaultLanguage              = "ID"
	defaultCurrency              = "IDR"
	defaultAccountID             = "0"
	defaultIsVerifiedPhoneNumber = "false"

	channelDesktop        = "DESKTOP"
	channelMobile         = "MOBILE"
	channelAndroid        = "ANDROID"
	channelIOS            = "IOS"
	channelWebview        = "WEBVIEW"
	channelWebviewUnknown = "WEBVIEW_UNKNOWN"

	b2bAffiliateResellerType = "B2B_AFFILIATE"

	applicationJSONContentType      = "application/json; charset=utf-8"
	defaultMaxClientRequestIDLength = 13

	headerKeyAuthorization = "Authorization"
	headerKeyAccessToken   = "accessToken"
	headerKeyTixToken      = "tixtoken"
	headerKeyTixSession    = "tixsession"
	cookieKeySessionID     = "PHPSESSID"

	localhostIpv4 = "127.0.0.1"
	localhostIpv6 = "0:0:0:0:0:0:0:1"
)

func setMandatoryHeaders(request *http.Request) http.Header {
	header := request.Header.Clone()

	// request ID
	header.Set(headerKeyRequestID, getRequestID(request.Header))

	// service ID
	header.Set(headerKeyServiceID, serviceID)

	// username
	header.Set(headerKeyUsername, defaultUsername)

	// store ID
	header.Set(headerKeyStoreID, getStoreID(request.Header))

	// account ID
	header.Set(headerKeyAccountID, defaultAccountID)

	// language
	header.Set(headerKeyLanguage, getLanguage(request.Header))

	// currency
	header.Set(headerKeyCurrency, getCurrency(request.Header))

	// device ID
	header.Set(headerKeyDeviceID, getDeviceID(request))

	// identity
	header.Set(headerKeyIdentity, getIdentity(request))

	// reseller ID
	resellerID := getResellerID(request)
	header.Set(headerKeyResellerID, resellerID)

	// reseller type
	header.Set(headerKeyResellerType, getResellerType(resellerID))

	// is verified phone number
	header.Set(headerKeyIsVerifiedPhoneNumber, defaultIsVerifiedPhoneNumber)

	// tag aff ID
	header.Set(headerKeyTagAffId, getTagAffID(request))

	clientIp := getClientIp(request)
	header.Set(headerKeyForwardedFor, clientIp)
	header.Set(headerKeyTrueClientIP, clientIp)

	// channel ID
	header.Set(headerKeyChannelID, getChannelID(request.Header))

	// content type
	header.Set(headerKeyContentType, getContentType(request.Header))

	sessionData, ok := request.Context().Value(memberAuthSessionDataKey).(memberAuthSessionData)
	if ok {
		setSessionHeaders(header, sessionData)

		// TODO: Remove this old auth keys after confirming not used by any service
		header.Set(headerKeyTixToken, sessionData.Token)
		header.Set(headerKeyTixSession, findCookie(request, cookieKeySessionID))
	}

	setOldKeys(header)

	header.Del(headerKeyCookie)

	return header
}

func findFirstNotEmptyHeader(header http.Header, headerKeys ...string) string {
	for _, headerKey := range headerKeys {
		if headerVal := header.Get(headerKey); headerVal != "" {
			return headerVal
		}
	}

	return ""
}

func findCookie(request *http.Request, name string) string {
	if v, err := request.Cookie(name); err == nil && v.Value != "" {
		return v.Value
	}

	return ""
}

func getRequestID(header http.Header) string {
	var requestID uuid.UUID

	requestID, err := uuid.NewV7()
	if err != nil {
		requestID, err = uuid.NewV4()
		if err != nil {
			logger.Warning("HTTP_REQUEST_ID", fmt.Sprintf("Failed to generate uuid v4, error: %s", err.Error()))
		}
	}

	clientRequestID := findFirstNotEmptyHeader(header, headerKeyRequestID, oldHeaderKeyRequestId)
	if clientRequestID != "" && clientRequestID != "NONE" {
		// generating maximum 50 characters in total
		clientRequestIDLength := len(clientRequestID)
		if clientRequestIDLength < defaultMaxClientRequestIDLength {
			return fmt.Sprintf("%s.%s", clientRequestID, requestID.String())
		}

		return fmt.Sprintf("%s.%s", clientRequestID[:defaultMaxClientRequestIDLength], requestID.String())
	}

	return requestID.String()
}

func getStoreID(header http.Header) string {
	storeID := findFirstNotEmptyHeader(header, headerKeyStoreID, oldHeaderKeyStoreId)
	if storeID == "" {
		return defaultStoreID
	}

	return storeID
}

func getLanguage(header http.Header) string {
	language := findFirstNotEmptyHeader(header, headerKeyLanguage, oldHeaderKeyLang, oldHeaderKeyLanguage)
	if language == "" {
		return defaultLanguage
	}

	return strings.ToUpper(language)
}

func getCurrency(header http.Header) string {
	currency := findFirstNotEmptyHeader(header, headerKeyCurrency, oldHeaderKeyCurrency)
	if currency == "" {
		return defaultCurrency
	}

	return strings.ToUpper(currency)
}

func getDeviceID(request *http.Request) string {
	if deviceID := findFirstNotEmptyHeader(request.Header, headerKeyDeviceID, oldHeaderKeyDeviceID); deviceID != "" {
		return deviceID
	}

	return findCookie(request, oldHeaderKeyDeviceID)
}

func getIdentity(request *http.Request) string {
	if deviceID := findFirstNotEmptyHeader(request.Header, headerKeyDeviceID, oldHeaderKeyDeviceID); deviceID != "" {
		return deviceID
	}

	if deviceID := findCookie(request, oldHeaderKeyDeviceID); deviceID != "" {
		return deviceID
	}

	return findCookie(request, headerKeySessionID)
}

func getResellerID(request *http.Request) string {
	if resellerID := findFirstNotEmptyHeader(request.Header, headerKeyResellerID, oldHeaderKeyResellerId); resellerID != "" {
		return resellerID
	}

	return findCookie(request, headerKeyTwh)
}

func getResellerType(resellerID string) string {
	resellerType := ""
	if resellerID != "" {
		nativeAppsResellerIds := []string{} // TODO: Load from config
		if !slices.Contains(nativeAppsResellerIds, resellerID) {
			resellerType = b2bAffiliateResellerType
		}
	}

	return resellerType
}

func getTagAffID(request *http.Request) string {
	if tagAffID := request.Header.Get(headerKeyTagAffId); tagAffID != "" {
		return tagAffID
	}

	return request.URL.Query().Get("tag_aff_id")
}

func getChannelID(header http.Header) string {
	channelIdV2 := header.Get(headerKeyChannelIDV2)
	platformV2 := header.Get(headerKeyPlatformV2)
	userAgent := header.Get(headerKeyUserAgent)

	if channelIdV2 == channelWebview {
		switch platformV2 {
		case channelAndroid:
			return channelAndroid
		case channelIOS:
			return channelIOS
		}

		return channelWebviewUnknown
	}

	if userAgent != "" {
		parser := gouseragent.NewParser()
		ua := parser.Parse(userAgent)
		if ua.IsDesktop() {
			return channelDesktop
		}
		if ua.IsMobile() {
			return channelMobile
		}
	}

	return defaultUnknown
}

func getContentType(header http.Header) string {
	if contentType := header.Get(headerKeyContentType); contentType != "" {
		return contentType
	}

	return applicationJSONContentType
}

func setSessionHeaders(header http.Header, sessionData memberAuthSessionData) {
	if sessionData.AccountID != 0 {
		header.Set(headerKeyAccountID, strconv.FormatInt(sessionData.AccountID, 10))
	}

	if sessionData.Username != "" {
		header.Set(headerKeyUsername, sessionData.Username)
	}

	if sessionData.BusinessID != "" {
		header.Set(headerKeyBusinessID, sessionData.BusinessID)
	}

	if sessionData.DeviceID != "" {
		header.Set(headerKeyDeviceID, sessionData.DeviceID)
	}

	if sessionData.Role != "" {
		header.Set(headerKeyRoleID, sessionData.Role)
	}

	if sessionData.LoginMedia != "" {
		header.Set(headerKeyLoginMedia, sessionData.LoginMedia)
	}

	if sessionData.LoyaltyLevel.Key != "" {
		header.Set(headerKeyLoyaltyLevel, sessionData.LoyaltyLevel.Key)
	}

	header.Set(headerKeyIsVerifiedPhoneNumber, strconv.FormatBool(sessionData.VerifiedPhoneNumber))

	header.Set(headerKeyAuthorization, sessionData.Token)
}

// TODO: Remove this function after all services are migrated to the new keys
func setOldKeys(header http.Header) {
	header.Set(oldHeaderKeyUsername, header.Get(headerKeyUsername))
	header.Set(oldHeaderKeyResellerId, header.Get(headerKeyResellerID))
	if header.Get(oldHeaderKeyUserAgent) == "" {
		header.Set(oldHeaderKeyUserAgent, header.Get(headerKeyUserAgent))
	}
}

func getClientIp(request *http.Request) string {
	var ipAddr string
	ipSources := []string{headerKeyTrueClientIP, headerKeyForwardedFor, headerKeyProxyClientIP, headerKeyRealIp}
	for _, source := range ipSources {
		ipAddr = request.Header.Get(source)
		if ipAddr != "" && !strings.EqualFold(ipAddr, defaultUnknown) {
			break
		}
	}

	if ipAddr == "" || strings.EqualFold(ipAddr, defaultUnknown) {
		ip, _, err := net.SplitHostPort(request.RemoteAddr)
		if err != nil {
			return ipAddr
		}
		if localhostIpv4 == ip || localhostIpv6 == ip {
			ipAddr = getLocalIP()
		}
	}

	if ipAddr != "" {
		if idx := strings.Index(ipAddr, ","); idx > 0 {
			ipAddr = strings.TrimSpace(ipAddr[:idx])
		}
	}

	return ipAddr
}

func getLocalIP() string {
	addr, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, address := range addr {
		// check the address type and if it is not a loopback the display it
		if inet, ok := address.(*net.IPNet); ok && !inet.IP.IsLoopback() {
			if inet.IP.To4() != nil {
				return inet.IP.String()
			}
		}
	}

	return ""
}

package middleware

import (
	"fmt"
	"net/http"

	"github.com/tiket/angga-kesuma/pkg/plugin/config"
	"github.com/tiket/angga-kesuma/pkg/plugin/outbound"
)

var MemberAuthName string = "member_auth"

func NewMemberAuth(cfg *config.AppConfig) *MemberAuth {
	memberAuth := &MemberAuth{}

	memberAuth.Cfg = cfg
	memberAuth.MemberSession = outbound.GetOutboundMemberSession(cfg)

	return memberAuth
}

type MemberAuth struct {
	Cfg           *config.AppConfig
	MemberSession *outbound.OutboundMemberSession
}

func (o *MemberAuth) Run(next http.Handler, pluginsConfig *config.EndpointConfig) (http.Handler, error) {

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		fmt.Println("member-auth plugin")

		next.ServeHTTP(w, req)
	}), nil
}

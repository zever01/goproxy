package auth

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/elazarl/goproxy"
)

var unauthorizedMsg = []byte("407 Proxy Authentication Required")

// AuthWithAddrFunc is an external authenticator contract.
// Context fields will be updated according to returned values.
type AuthWithAddrFunc func(
	RemoteAddr string,
	user string,
	passwd string,
) (
	updatedRemoteAddr string,
	updatedUser string,
	updatedPasswd string,
	ok bool,
)

func BasicUnauthorized(req *http.Request, realm string) *http.Response {
	// TODO(elazar): verify realm is well formed
	return &http.Response{
		StatusCode: 407,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Request:    req,
		Header: http.Header{
			"Proxy-Authenticate": []string{"Basic realm=" + realm},
			"Proxy-Connection":   []string{"close"},
		},
		Body:          ioutil.NopCloser(bytes.NewBuffer(unauthorizedMsg)),
		ContentLength: int64(len(unauthorizedMsg)),
	}
}

var proxyAuthorizationHeader = "Proxy-Authorization"

func auth(req *http.Request, f AuthWithAddrFunc) (string, string, string, bool) {
	authheader := strings.SplitN(req.Header.Get(proxyAuthorizationHeader), " ", 2)
	req.Header.Del(proxyAuthorizationHeader)
	if len(authheader) != 2 || authheader[0] != "Basic" {
		return "", "", "", false
	}
	userpassraw, err := base64.StdEncoding.DecodeString(authheader[1])
	if err != nil {
		return "", "", "", false
	}
	userpass := strings.SplitN(string(userpassraw), ":", 2)
	if len(userpass) != 2 {
		return "", "", "", false
	}

	return f(req.RemoteAddr, userpass[0], userpass[1])
}

// Basic returns a basic HTTP authentication handler for requests
//
// You probably want to use auth.ProxyBasic(proxy) to enable authentication for all proxy activities
func Basic(realm string, f func(user, passwd string) bool) goproxy.ReqHandler {
	return BasicWithAddr(
		realm,
		func(
			RemoteAddr string,
			user string,
			passwd string,
		) (string, string, string, bool) {
			return RemoteAddr, user, passwd, f(user, passwd)
		},
	)
}

// BasicWithAddr returns a basic HTTP authentication handler for requests
//
// You probably want to use auth.ProxyBasicWithAddrWithAddr(proxy) to enable authentication for all proxy activities
func BasicWithAddr(realm string, f AuthWithAddrFunc) goproxy.ReqHandler {
	return goproxy.FuncReqHandler(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		remoteAddr, user, pass, ok := auth(req, f)
		if !ok {
			return nil, BasicUnauthorized(req, realm)
		}

		ctx.Req.RemoteAddr = remoteAddr
		ctx.User = user
		ctx.Password = pass

		return req, nil
	})
}

// BasicConnect returns a basic HTTP authentication handler for CONNECT requests
//
// You probably want to use auth.ProxyBasic(proxy) to enable authentication for all proxy activities
func BasicConnect(realm string, f func(user, passwd string) bool) goproxy.HttpsHandler {
	return BasicConnectWithAddr(
		realm,
		func(
			RemoteAddr string,
			user string,
			passwd string,
		) (string, string, string, bool) {
			return RemoteAddr, user, passwd, f(user, passwd)
		},
	)
}

// BasicConnectWithAddr returns a basic HTTP authentication handler for CONNECT requests
//
// You probably want to use auth.ProxyBasicWithAddr(proxy) to enable authentication for all proxy activities
func BasicConnectWithAddr(realm string, f AuthWithAddrFunc) goproxy.HttpsHandler {
	return goproxy.FuncHttpsHandler(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		remoteAddr, user, pass, ok := auth(ctx.Req, f)
		if !ok {
			ctx.Resp = BasicUnauthorized(ctx.Req, realm)
			return goproxy.RejectConnect, host
		}

		ctx.Req.RemoteAddr = remoteAddr
		ctx.User = user
		ctx.Password = pass

		return goproxy.OkConnect, host
	})
}

// ProxyBasic will force HTTP authentication before any request to the proxy is processed
func ProxyBasic(proxy *goproxy.ProxyHttpServer, realm string, f func(user, passwd string) bool) {
	proxy.OnRequest().Do(Basic(realm, f))
	proxy.OnRequest().HandleConnect(BasicConnect(realm, f))
}

// ProxyBasic will force HTTP authentication before any request to the proxy is processed
func ProxyBasicWithAddr(proxy *goproxy.ProxyHttpServer, realm string, f AuthWithAddrFunc) {
	proxy.OnRequest().Do(BasicWithAddr(realm, f))
	proxy.OnRequest().HandleConnect(BasicConnectWithAddr(realm, f))
}
